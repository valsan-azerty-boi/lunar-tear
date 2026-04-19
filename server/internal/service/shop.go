package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"

	"google.golang.org/protobuf/types/known/emptypb"
)

var shopDiffTables = []string{
	"IUserShopItem",
	"IUserShopReplaceable",
	"IUserShopReplaceableLineup",
	"IUserGem",
	"IUserConsumableItem",
	"IUserMaterial",
	"IUserImportantItem",
	"IUserPremiumItem",
	"IUserStatus",
	"IUserCostume",
	"IUserCostumeActiveSkill",
	"IUserCharacter",
	"IUserWeapon",
	"IUserWeaponSkill",
	"IUserWeaponAbility",
	"IUserWeaponNote",
}

type ShopServiceServer struct {
	pb.UnimplementedShopServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.ShopCatalog
	granter  *store.PossessionGranter
}

func NewShopServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.ShopCatalog, granter *store.PossessionGranter) *ShopServiceServer {
	return &ShopServiceServer{users: users, sessions: sessions, catalog: catalog, granter: granter}
}

func (s *ShopServiceServer) Buy(ctx context.Context, req *pb.BuyRequest) (*pb.BuyResponse, error) {
	log.Printf("[ShopService] Buy: shopId=%d items=%v", req.ShopId, req.ShopItems)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	snapshot, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		for shopItemId, qty := range req.ShopItems {
			item, ok := s.catalog.Items[shopItemId]
			if !ok {
				log.Printf("[ShopService] Buy: unknown shopItemId=%d, skipping", shopItemId)
				continue
			}

			totalPrice := item.Price * qty
			if err := store.DeductPrice(user, item.PriceType, item.PriceId, totalPrice); err != nil {
				log.Printf("[ShopService] Buy: deduct failed shopItemId=%d: %v", shopItemId, err)
				continue
			}

			for _, content := range s.catalog.Contents[shopItemId] {
				s.granter.GrantFull(user,
					model.PossessionType(content.PossessionType),
					content.PossessionId,
					content.Count*qty,
					nowMillis,
				)
			}

			s.applyContentEffects(user, shopItemId, qty, nowMillis)

			si := user.ShopItems[shopItemId]
			si.ShopItemId = shopItemId
			si.BoughtCount += qty
			si.LatestBoughtCountChangedDatetime = nowMillis
			si.LatestVersion = nowMillis
			user.ShopItems[shopItemId] = si
		}
	})
	if err != nil {
		return nil, fmt.Errorf("shop buy: %w", err)
	}

	diff := userdata.BuildDiffFromTables(userdata.ProjectTables(snapshot, shopDiffTables))
	userdata.AddWeaponStoryDiff(diff, snapshot, s.granter.DrainChangedStoryWeaponIds())

	return &pb.BuyResponse{
		OverflowPossession: []*pb.Possession{},
		DiffUserData:       diff,
	}, nil
}

func (s *ShopServiceServer) RefreshUserData(ctx context.Context, req *pb.RefreshRequest) (*pb.RefreshResponse, error) {
	log.Printf("[ShopService] RefreshUserData: isGemUsed=%v", req.IsGemUsed)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	snapshot, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		if len(user.ShopReplaceableLineup) == 0 && len(s.catalog.ItemShopPool) > 0 {
			for i, itemId := range s.catalog.ItemShopPool {
				slot := int32(i + 1)
				user.ShopReplaceableLineup[slot] = store.UserShopReplaceableLineupState{
					SlotNumber:    slot,
					ShopItemId:    itemId,
					LatestVersion: nowMillis,
				}
			}
		}
		if req.IsGemUsed {
			user.ShopReplaceable.LineupUpdateCount++
			user.ShopReplaceable.LatestLineupUpdateDatetime = nowMillis
			for _, itemId := range s.catalog.ItemShopPool {
				if si, ok := user.ShopItems[itemId]; ok {
					si.BoughtCount = 0
					si.LatestVersion = nowMillis
					user.ShopItems[itemId] = si
				}
			}
		}
	})
	if err != nil {
		return nil, fmt.Errorf("shop refresh: %w", err)
	}

	diff := userdata.BuildDiffFromTables(userdata.ProjectTables(snapshot, shopDiffTables))

	return &pb.RefreshResponse{
		DiffUserData: diff,
	}, nil
}

func (s *ShopServiceServer) GetCesaLimit(_ context.Context, _ *emptypb.Empty) (*pb.GetCesaLimitResponse, error) {
	log.Printf("[ShopService] GetCesaLimit")
	return &pb.GetCesaLimitResponse{
		CesaLimit:    []*pb.CesaLimit{},
		DiffUserData: userdata.EmptyDiff(),
	}, nil
}

func (s *ShopServiceServer) CreatePurchaseTransaction(ctx context.Context, req *pb.CreatePurchaseTransactionRequest) (*pb.CreatePurchaseTransactionResponse, error) {
	log.Printf("[ShopService] CreatePurchaseTransaction: shopId=%d shopItemId=%d productId=%s",
		req.ShopId, req.ShopItemId, req.ProductId)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	snapshot, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		item, ok := s.catalog.Items[req.ShopItemId]
		if !ok {
			log.Printf("[ShopService] CreatePurchaseTransaction: unknown shopItemId=%d", req.ShopItemId)
			return
		}

		if err := store.DeductPrice(user, item.PriceType, item.PriceId, item.Price); err != nil {
			log.Printf("[ShopService] CreatePurchaseTransaction: deduct failed: %v", err)
		}

		for _, content := range s.catalog.Contents[req.ShopItemId] {
			s.granter.GrantFull(user,
				model.PossessionType(content.PossessionType),
				content.PossessionId,
				content.Count,
				nowMillis,
			)
		}

		s.applyContentEffects(user, req.ShopItemId, 1, nowMillis)

		si := user.ShopItems[req.ShopItemId]
		si.ShopItemId = req.ShopItemId
		si.BoughtCount++
		if item.ShopItemLimitedStockId > 0 {
			if maxCount, ok := s.catalog.LimitedStock[item.ShopItemLimitedStockId]; ok && si.BoughtCount >= maxCount {
				si.BoughtCount = 0
			}
		}
		si.LatestBoughtCountChangedDatetime = nowMillis
		si.LatestVersion = nowMillis
		user.ShopItems[req.ShopItemId] = si
	})
	if err != nil {
		return nil, fmt.Errorf("create purchase transaction: %w", err)
	}

	txId := fmt.Sprintf("tx_%d_%d_%d", userId, req.ShopItemId, nowMillis)

	diff := userdata.BuildDiffFromTables(userdata.ProjectTables(snapshot, shopDiffTables))

	return &pb.CreatePurchaseTransactionResponse{
		PurchaseTransactionId: txId,
		DiffUserData:          diff,
	}, nil
}

func (s *ShopServiceServer) PurchaseGooglePlayStoreProduct(ctx context.Context, req *pb.PurchaseGooglePlayStoreProductRequest) (*pb.PurchaseGooglePlayStoreProductResponse, error) {
	log.Printf("[ShopService] PurchaseGooglePlayStoreProduct: txId=%s", req.PurchaseTransactionId)

	userId := currentUserId(ctx, s.users, s.sessions)
	snapshot, err := s.users.LoadUser(userId)
	if err != nil {
		return nil, fmt.Errorf("purchase google play: %w", err)
	}

	diff := userdata.BuildDiffFromTables(userdata.ProjectTables(snapshot, shopDiffTables))

	return &pb.PurchaseGooglePlayStoreProductResponse{
		OverflowPossession: []*pb.Possession{},
		DiffUserData:       diff,
	}, nil
}

func (s *ShopServiceServer) applyContentEffects(user *store.UserState, shopItemId, qty int32, nowMillis int64) {
	for _, effect := range s.catalog.Effects[shopItemId] {
		switch effect.EffectTargetType {
		case model.EffectTargetStaminaRecovery:
			maxMillis := s.catalog.MaxStaminaMillis[user.Status.Level]
			millis := s.resolveEffectMillis(effect.EffectValueType, effect.EffectValue, user.Status.Level)
			store.RecoverStamina(user, millis*qty, maxMillis, nowMillis)
		default:
			log.Printf("[ShopService] unhandled effect: shopItemId=%d targetType=%d", shopItemId, effect.EffectTargetType)
		}
	}
}

func (s *ShopServiceServer) resolveEffectMillis(effectValueType, effectValue, userLevel int32) int32 {
	switch effectValueType {
	case model.EffectValueFixed:
		return effectValue
	case model.EffectValuePermil:
		maxMillis := s.catalog.MaxStaminaMillis[userLevel]
		return effectValue * maxMillis / 1000
	default:
		return 0
	}
}
