package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/store"
)

const partsMaxLevel = int32(15)

type PartsServiceServer struct {
	pb.UnimplementedPartsServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.PartsCatalog
	config   *masterdata.GameConfig
}

func NewPartsServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.PartsCatalog, config *masterdata.GameConfig) *PartsServiceServer {
	return &PartsServiceServer{users: users, sessions: sessions, catalog: catalog, config: config}
}

func (s *PartsServiceServer) Sell(ctx context.Context, req *pb.PartsSellRequest) (*pb.PartsSellResponse, error) {
	log.Printf("[PartsService] Sell: %d part(s)", len(req.UserPartsUuid))

	userId := CurrentUserId(ctx, s.users, s.sessions)

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		totalGold := int32(0)
		for _, uuid := range req.UserPartsUuid {
			part, ok := user.Parts[uuid]
			if !ok {
				log.Printf("[PartsService] Sell: uuid=%s not found, skipping", uuid)
				continue
			}
			if part.IsProtected {
				log.Printf("[PartsService] Sell: uuid=%s is protected, skipping", uuid)
				continue
			}

			partDef, ok := s.catalog.PartsById[part.PartsId]
			if !ok {
				log.Printf("[PartsService] Sell: partsId=%d not in catalog, skipping", part.PartsId)
				continue
			}

			sellFunc, ok := s.catalog.SellPriceByRarity[partDef.RarityType]
			if !ok {
				log.Printf("[PartsService] Sell: no sell price func for rarity=%d, skipping", partDef.RarityType)
				continue
			}

			gold := sellFunc.Evaluate(part.Level)
			totalGold += gold
			delete(user.Parts, uuid)
			for k := range user.PartsStatusSubs {
				if k.UserPartsUuid == uuid {
					delete(user.PartsStatusSubs, k)
				}
			}
			log.Printf("[PartsService] Sell: uuid=%s partsId=%d level=%d -> %d gold", uuid, part.PartsId, part.Level, gold)
		}

		if totalGold > 0 {
			user.ConsumableItems[s.config.ConsumableItemIdForGold] += totalGold
			log.Printf("[PartsService] Sell: total gold +%d", totalGold)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("parts sell: %w", err)
	}

	return &pb.PartsSellResponse{}, nil
}

func (s *PartsServiceServer) Enhance(ctx context.Context, req *pb.PartsEnhanceRequest) (*pb.PartsEnhanceResponse, error) {
	log.Printf("[PartsService] Enhance: uuid=%s", req.UserPartsUuid)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	isSuccess := false

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		part, ok := user.Parts[req.UserPartsUuid]
		if !ok {
			log.Printf("[PartsService] Enhance: part uuid=%s not found", req.UserPartsUuid)
			return
		}

		if part.Level >= partsMaxLevel {
			log.Printf("[PartsService] Enhance: part uuid=%s already at max level %d", req.UserPartsUuid, part.Level)
			return
		}

		partDef, ok := s.catalog.PartsById[part.PartsId]
		if !ok {
			log.Printf("[PartsService] Enhance: part master id=%d not found", part.PartsId)
			return
		}

		rarity, ok := s.catalog.RarityByRarityType[partDef.RarityType]
		if !ok {
			log.Printf("[PartsService] Enhance: rarity type=%d not found", partDef.RarityType)
			return
		}

		goldCost := int32(0)
		if prices, ok := s.catalog.PriceByGroupAndLevel[rarity.PartsLevelUpPriceGroupId]; ok {
			goldCost = prices[part.Level]
		}

		currentGold := user.ConsumableItems[s.config.ConsumableItemIdForGold]
		if currentGold < goldCost {
			log.Printf("[PartsService] Enhance: insufficient gold have=%d need=%d", currentGold, goldCost)
			return
		}

		user.ConsumableItems[s.config.ConsumableItemIdForGold] -= goldCost

		successRate := int32(1000)
		if rates, ok := s.catalog.RateByGroupAndLevel[rarity.PartsLevelUpRateGroupId]; ok {
			if r, ok := rates[part.Level]; ok {
				successRate = r
			}
		}

		if rand.Intn(1000) < int(successRate) {
			part.Level++
			isSuccess = true
			log.Printf("[PartsService] Enhance: SUCCESS partsId=%d level %d -> %d (rate=%d‰, cost=%d gold)",
				part.PartsId, part.Level-1, part.Level, successRate, goldCost)

			s.grantSubStatuses(user, req.UserPartsUuid, part, partDef, nowMillis)
		} else {
			log.Printf("[PartsService] Enhance: FAIL partsId=%d stays level %d (rate=%d‰, cost=%d gold)",
				part.PartsId, part.Level, successRate, goldCost)
		}

		part.LatestVersion = nowMillis
		user.Parts[req.UserPartsUuid] = part
	})
	if err != nil {
		return nil, fmt.Errorf("parts enhance: %w", err)
	}

	return &pb.PartsEnhanceResponse{
		IsSuccess: isSuccess,
	}, nil
}

func (s *PartsServiceServer) grantSubStatuses(user *store.UserState, uuid string, part store.PartsState, partDef masterdata.EntityMParts, nowMillis int64) {
	unlockLevels := s.catalog.SubStatusUnlockLvls[partDef.RarityType]
	pool := s.catalog.SubStatusPool[partDef.PartsStatusSubLotteryGroupId]
	if len(pool) == 0 {
		return
	}

	for slotIdx, lvl := range unlockLevels {
		if part.Level != lvl {
			continue
		}
		statusIndex := int32(slotIdx + 1)
		key := store.PartsStatusSubKey{UserPartsUuid: uuid, StatusIndex: statusIndex}
		if _, exists := user.PartsStatusSubs[key]; exists {
			continue
		}

		pick := pool[rand.Intn(len(pool))]
		def, ok := s.catalog.PartsStatusMainById[pick]
		if !ok {
			continue
		}

		statusValue := def.StatusChangeInitialValue
		if f, ok := s.catalog.FuncResolver.Resolve(def.StatusNumericalFunctionId); ok {
			statusValue = f.Evaluate(part.Level)
		}

		user.PartsStatusSubs[key] = store.PartsStatusSubState{
			UserPartsUuid:           uuid,
			StatusIndex:             statusIndex,
			PartsStatusSubLotteryId: pick,
			Level:                   part.Level,
			StatusKindType:          def.StatusKindType,
			StatusCalculationType:   def.StatusCalculationType,
			StatusChangeValue:       statusValue,
			LatestVersion:           nowMillis,
		}
		log.Printf("[PartsService] Enhance: granted sub-status slot=%d lotteryId=%d kind=%d calc=%d val=%d",
			statusIndex, pick, def.StatusKindType, def.StatusCalculationType, statusValue)
	}
}

func (s *PartsServiceServer) ReplacePreset(ctx context.Context, req *pb.PartsReplacePresetRequest) (*pb.PartsReplacePresetResponse, error) {
	log.Printf("[PartsService] ReplacePreset: preset=%d uuids=[%s, %s, %s]",
		req.UserPartsPresetNumber, req.UserPartsUuid01, req.UserPartsUuid02, req.UserPartsUuid03)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		preset := user.PartsPresets[req.UserPartsPresetNumber]
		preset.UserPartsPresetNumber = req.UserPartsPresetNumber
		preset.UserPartsUuid01 = req.UserPartsUuid01
		preset.UserPartsUuid02 = req.UserPartsUuid02
		preset.UserPartsUuid03 = req.UserPartsUuid03
		preset.LatestVersion = nowMillis
		user.PartsPresets[req.UserPartsPresetNumber] = preset
	})
	if err != nil {
		return nil, fmt.Errorf("parts replace preset: %w", err)
	}

	return &pb.PartsReplacePresetResponse{}, nil
}
