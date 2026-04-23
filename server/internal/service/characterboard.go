package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

type CharacterBoardServiceServer struct {
	pb.UnimplementedCharacterBoardServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.CharacterBoardCatalog
}

func NewCharacterBoardServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.CharacterBoardCatalog) *CharacterBoardServiceServer {
	return &CharacterBoardServiceServer{users: users, sessions: sessions, catalog: catalog}
}

func (s *CharacterBoardServiceServer) ReleasePanel(ctx context.Context, req *pb.ReleasePanelRequest) (*pb.ReleasePanelResponse, error) {
	log.Printf("[CharacterBoardService] ReleasePanel: panelIds=%v", req.CharacterBoardPanelId)

	userId := CurrentUserId(ctx, s.users, s.sessions)

	s.users.UpdateUser(userId, func(user *store.UserState) {
		for _, panelId := range req.CharacterBoardPanelId {
			panel, ok := s.catalog.PanelById[panelId]
			if !ok {
				log.Printf("[CharacterBoardService] unknown panelId=%d, skipping", panelId)
				continue
			}

			s.consumeCosts(user, panel)
			s.setReleaseBit(user, panel)
			s.applyEffects(user, panel)
		}
	})

	return &pb.ReleasePanelResponse{}, nil
}

func (s *CharacterBoardServiceServer) consumeCosts(user *store.UserState, panel masterdata.EntityMCharacterBoardPanel) {
	costs := s.catalog.ReleaseCostsByGroupId[panel.CharacterBoardPanelReleasePossessionGroupId]
	for _, cost := range costs {
		store.DeductPossession(user, model.PossessionType(cost.PossessionType), cost.PossessionId, cost.Count)
	}
}

func (s *CharacterBoardServiceServer) setReleaseBit(user *store.UserState, panel masterdata.EntityMCharacterBoardPanel) {
	boardId := panel.CharacterBoardId
	board := user.CharacterBoards[boardId]
	board.CharacterBoardId = boardId

	bitFieldIndex := (panel.SortOrder - 1) / 32
	bitPosition := (panel.SortOrder - 1) % 32
	mask := int32(1 << uint(bitPosition))

	switch bitFieldIndex {
	case 0:
		board.PanelReleaseBit1 |= mask
	case 1:
		board.PanelReleaseBit2 |= mask
	case 2:
		board.PanelReleaseBit3 |= mask
	case 3:
		board.PanelReleaseBit4 |= mask
	}

	user.CharacterBoards[boardId] = board
}

func (s *CharacterBoardServiceServer) applyEffects(user *store.UserState, panel masterdata.EntityMCharacterBoardPanel) {
	effects := s.catalog.ReleaseEffectsByGroupId[panel.CharacterBoardPanelReleaseEffectGroupId]
	for _, eff := range effects {
		switch model.CharacterBoardEffectType(eff.CharacterBoardEffectType) {
		case model.CharacterBoardEffectTypeAbility:
			s.applyAbilityEffect(user, eff)
		case model.CharacterBoardEffectTypeStatusUp:
			s.applyStatusUpEffect(user, eff)
		}
	}
}

func (s *CharacterBoardServiceServer) applyAbilityEffect(user *store.UserState, eff masterdata.EntityMCharacterBoardPanelReleaseEffectGroup) {
	ability, ok := s.catalog.AbilityById[eff.CharacterBoardEffectId]
	if !ok {
		log.Printf("[CharacterBoardService] unknown abilityId=%d", eff.CharacterBoardEffectId)
		return
	}

	characterId := s.resolveCharacterId(ability.CharacterBoardEffectTargetGroupId)
	if characterId == 0 {
		return
	}

	key := store.CharacterBoardAbilityKey{CharacterId: characterId, AbilityId: ability.AbilityId}
	state := user.CharacterBoardAbilities[key]
	state.CharacterId = characterId
	state.AbilityId = ability.AbilityId
	state.Level += eff.EffectValue

	if maxLvl, ok := s.catalog.AbilityMaxLevel[key]; ok && state.Level > maxLvl {
		state.Level = maxLvl
	}

	user.CharacterBoardAbilities[key] = state
}

func (s *CharacterBoardServiceServer) applyStatusUpEffect(user *store.UserState, eff masterdata.EntityMCharacterBoardPanelReleaseEffectGroup) {
	statusUp, ok := s.catalog.StatusUpById[eff.CharacterBoardEffectId]
	if !ok {
		log.Printf("[CharacterBoardService] unknown statusUpId=%d", eff.CharacterBoardEffectId)
		return
	}

	characterId := s.resolveCharacterId(statusUp.CharacterBoardEffectTargetGroupId)
	if characterId == 0 {
		return
	}

	supType := model.CharacterBoardStatusUpType(statusUp.CharacterBoardStatusUpType)
	calcType := model.StatusUpTypeToCalcType(supType)

	key := store.CharacterBoardStatusUpKey{
		CharacterId:           characterId,
		StatusCalculationType: int32(calcType),
	}
	state := user.CharacterBoardStatusUps[key]
	state.CharacterId = characterId
	state.StatusCalculationType = int32(calcType)

	switch supType {
	case model.CharacterBoardStatusUpTypeAgilityAdd, model.CharacterBoardStatusUpTypeAgilityMultiply:
		state.Agility += eff.EffectValue
	case model.CharacterBoardStatusUpTypeAttackAdd, model.CharacterBoardStatusUpTypeAttackMultiply:
		state.Attack += eff.EffectValue
	case model.CharacterBoardStatusUpTypeCritAttackAdd:
		state.CriticalAttack += eff.EffectValue
	case model.CharacterBoardStatusUpTypeCritRatioAdd:
		state.CriticalRatio += eff.EffectValue
	case model.CharacterBoardStatusUpTypeHpAdd, model.CharacterBoardStatusUpTypeHpMultiply:
		state.Hp += eff.EffectValue
	case model.CharacterBoardStatusUpTypeVitalityAdd, model.CharacterBoardStatusUpTypeVitalityMultiply:
		state.Vitality += eff.EffectValue
	}

	user.CharacterBoardStatusUps[key] = state
}

func (s *CharacterBoardServiceServer) resolveCharacterId(targetGroupId int32) int32 {
	targets := s.catalog.EffectTargetsByGroupId[targetGroupId]
	for _, t := range targets {
		if t.TargetValue != 0 {
			return t.TargetValue
		}
	}
	log.Printf("[CharacterBoardService] no characterId resolved for targetGroupId=%d", targetGroupId)
	return 0
}
