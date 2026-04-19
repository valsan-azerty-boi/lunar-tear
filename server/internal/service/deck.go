package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"
)

type DeckServiceServer struct {
	pb.UnimplementedDeckServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
}

func NewDeckServiceServer(users store.UserRepository, sessions store.SessionRepository) *DeckServiceServer {
	return &DeckServiceServer{users: users, sessions: sessions}
}

func (s *DeckServiceServer) UpdateName(ctx context.Context, req *pb.UpdateNameRequest) (*pb.UpdateNameResponse, error) {
	log.Printf("[DeckService] UpdateName: deckType=%d deckNumber=%d name=%q", req.DeckType, req.UserDeckNumber, req.Name)
	userId := currentUserId(ctx, s.users, s.sessions)

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		deckKey := store.DeckKey{DeckType: model.DeckType(req.DeckType), UserDeckNumber: req.UserDeckNumber}
		deck := user.Decks[deckKey]
		deck.Name = req.Name
		user.Decks[deckKey] = deck
	})

	result := userdata.ProjectTables(user, []string{"IUserDeck"})
	return &pb.UpdateNameResponse{
		DiffUserData: userdata.BuildDiffFromTables(result),
	}, nil
}

func (s *DeckServiceServer) RefreshDeckPower(ctx context.Context, req *pb.RefreshDeckPowerRequest) (*pb.RefreshDeckPowerResponse, error) {
	log.Printf("[DeckService] RefreshDeckPower: deckType=%d deckNumber=%d", req.DeckType, req.UserDeckNumber)
	userId := currentUserId(ctx, s.users, s.sessions)

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		if req.DeckPower == nil {
			log.Printf("[DeckService] RefreshDeckPower: deckPower is nil")
			return
		}

		dt := model.DeckType(req.DeckType)
		deckKey := store.DeckKey{DeckType: dt, UserDeckNumber: req.UserDeckNumber}
		deck, ok := user.Decks[deckKey]
		if !ok {
			log.Fatalf("[DeckService] RefreshDeckPower: deck not found")
		}

		deck.Power = req.DeckPower.Power
		user.Decks[deckKey] = deck

		for _, cp := range []*pb.DeckCharacterPower{
			req.DeckPower.DeckCharacterPower01,
			req.DeckPower.DeckCharacterPower02,
			req.DeckPower.DeckCharacterPower03,
		} {
			if cp == nil || cp.UserDeckCharacterUuid == "" {
				continue
			}

			if dc, ok := user.DeckCharacters[cp.UserDeckCharacterUuid]; ok {
				dc.Power = cp.Power
				user.DeckCharacters[cp.UserDeckCharacterUuid] = dc
			}
		}

		note := user.DeckTypeNotes[dt]
		if req.DeckPower.Power > note.MaxDeckPower {
			note.DeckType = dt
			note.MaxDeckPower = req.DeckPower.Power
			user.DeckTypeNotes[dt] = note
		}
	})

	result := userdata.ProjectTables(user, []string{
		"IUserDeck", "IUserDeckCharacter", "IUserDeckTypeNote",
	})
	return &pb.RefreshDeckPowerResponse{
		DiffUserData: userdata.BuildDiffFromTables(result),
	}, nil
}

func (s *DeckServiceServer) RefreshMultiDeckPower(ctx context.Context, req *pb.RefreshMultiDeckPowerRequest) (*pb.RefreshMultiDeckPowerResponse, error) {
	log.Printf("[DeckService] RefreshMultiDeckPower: %d entries", len(req.DeckPowerInfo))
	userId := currentUserId(ctx, s.users, s.sessions)

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		for _, info := range req.DeckPowerInfo {
			if info.DeckPower == nil {
				continue
			}

			dt := model.DeckType(info.DeckType)
			deckKey := store.DeckKey{DeckType: dt, UserDeckNumber: info.UserDeckNumber}
			deck, ok := user.Decks[deckKey]
			if !ok {
				log.Printf("[DeckService] RefreshMultiDeckPower: deck not found deckType=%d deckNumber=%d", info.DeckType, info.UserDeckNumber)
				continue
			}

			deck.Power = info.DeckPower.Power
			user.Decks[deckKey] = deck

			for _, cp := range []*pb.DeckCharacterPower{
				info.DeckPower.DeckCharacterPower01,
				info.DeckPower.DeckCharacterPower02,
				info.DeckPower.DeckCharacterPower03,
			} {
				if cp == nil || cp.UserDeckCharacterUuid == "" {
					continue
				}
				if dc, ok := user.DeckCharacters[cp.UserDeckCharacterUuid]; ok {
					dc.Power = cp.Power
					user.DeckCharacters[cp.UserDeckCharacterUuid] = dc
				}
			}

			note := user.DeckTypeNotes[dt]
			if info.DeckPower.Power > note.MaxDeckPower {
				note.DeckType = dt
				note.MaxDeckPower = info.DeckPower.Power
				user.DeckTypeNotes[dt] = note
			}
		}
	})

	result := userdata.ProjectTables(user, []string{
		"IUserDeck", "IUserDeckCharacter", "IUserDeckTypeNote",
	})
	return &pb.RefreshMultiDeckPowerResponse{
		DiffUserData: userdata.BuildDiffFromTables(result),
	}, nil
}

func deckSlotsFromProto(deck *pb.Deck) []store.DeckCharacterInput {
	slots := make([]store.DeckCharacterInput, 3)
	for i, ch := range []*pb.DeckCharacter{deck.Character01, deck.Character02, deck.Character03} {
		if ch == nil {
			continue
		}
		slots[i] = store.DeckCharacterInput{
			UserCostumeUuid:    ch.UserCostumeUuid,
			MainUserWeaponUuid: ch.MainUserWeaponUuid,
			SubWeaponUuids:     ch.SubUserWeaponUuid,
			PartsUuids:         ch.UserPartsUuid,
			UserCompanionUuid:  ch.UserCompanionUuid,
			UserThoughtUuid:    ch.UserThoughtUuid,
			DressupCostumeId:   ch.DressupCostumeId,
		}
	}
	return slots
}

func (s *DeckServiceServer) ReplaceDeck(ctx context.Context, req *pb.ReplaceDeckRequest) (*pb.ReplaceDeckResponse, error) {
	log.Printf("[DeckService] ReplaceDeck: deckType=%d deckNumber=%d", req.DeckType, req.UserDeckNumber)
	if req.Deck != nil {
		for i, ch := range []*pb.DeckCharacter{req.Deck.Character01, req.Deck.Character02, req.Deck.Character03} {
			if ch == nil {
				continue
			}
			log.Printf("[DeckService] ReplaceDeck slot %d: costume=%s mainWeapon=%s subWeapons=%v companion=%s thought=%s",
				i+1, ch.UserCostumeUuid, ch.MainUserWeaponUuid, ch.SubUserWeaponUuid, ch.UserCompanionUuid, ch.UserThoughtUuid)
		}
	}
	userId := currentUserId(ctx, s.users, s.sessions)

	oldUser, _ := s.users.LoadUser(userId)
	tracker := userdata.NewDeleteTracker().
		Track("IUserDeckSubWeaponGroup", oldUser, userdata.DeckSubWeaponRecords,
			[]string{"userId", "userDeckCharacterUuid", "userWeaponUuid"}).
		Track("IUserDeckPartsGroup", oldUser, userdata.DeckPartsGroupRecords,
			[]string{"userId", "userDeckCharacterUuid", "userPartsUuid"}).
		Track("IUserDeckCharacterDressupCostume", oldUser, userdata.DeckDressupCostumeRecords,
			[]string{"userId", "userDeckCharacterUuid"})

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		if req.Deck == nil {
			return
		}
		store.ApplyDeckReplacement(user, model.DeckType(req.DeckType), req.UserDeckNumber, deckSlotsFromProto(req.Deck), gametime.NowMillis())
	})

	result := userdata.ProjectTables(user, []string{
		"IUserDeck", "IUserDeckCharacter", "IUserDeckSubWeaponGroup", "IUserDeckPartsGroup",
		"IUserDeckCharacterDressupCostume",
	})
	return &pb.ReplaceDeckResponse{
		DiffUserData: tracker.Apply(user, result),
	}, nil
}

func (s *DeckServiceServer) ReplaceTripleDeck(ctx context.Context, req *pb.ReplaceTripleDeckRequest) (*pb.ReplaceTripleDeckResponse, error) {
	log.Printf("[DeckService] ReplaceTripleDeck: deckType=%d deckNumber=%d", req.DeckType, req.UserDeckNumber)
	userId := currentUserId(ctx, s.users, s.sessions)

	oldUser, _ := s.users.LoadUser(userId)
	tracker := userdata.NewDeleteTracker().
		Track("IUserDeckSubWeaponGroup", oldUser, userdata.DeckSubWeaponRecords,
			[]string{"userId", "userDeckCharacterUuid", "userWeaponUuid"}).
		Track("IUserDeckPartsGroup", oldUser, userdata.DeckPartsGroupRecords,
			[]string{"userId", "userDeckCharacterUuid", "userPartsUuid"}).
		Track("IUserDeckCharacterDressupCostume", oldUser, userdata.DeckDressupCostumeRecords,
			[]string{"userId", "userDeckCharacterUuid"})

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		nowMillis := gametime.NowMillis()
		for idx, detail := range []*pb.DeckDetail{req.DeckDetail01, req.DeckDetail02, req.DeckDetail03} {
			if detail == nil || detail.Deck == nil {
				continue
			}
			log.Printf("[DeckService] ReplaceTripleDeck detail %d: deckType=%d deckNumber=%d", idx+1, detail.DeckType, detail.UserDeckNumber)
			if detail.Deck != nil {
				for i, ch := range []*pb.DeckCharacter{detail.Deck.Character01, detail.Deck.Character02, detail.Deck.Character03} {
					if ch == nil {
						continue
					}
					log.Printf("[DeckService] ReplaceTripleDeck detail %d slot %d: costume=%s mainWeapon=%s subWeapons=%v companion=%s thought=%s",
						idx+1, i+1, ch.UserCostumeUuid, ch.MainUserWeaponUuid, ch.SubUserWeaponUuid, ch.UserCompanionUuid, ch.UserThoughtUuid)
				}
			}
			store.ApplyDeckReplacement(user, model.DeckType(detail.DeckType), detail.UserDeckNumber, deckSlotsFromProto(detail.Deck), nowMillis)
		}
	})

	result := userdata.ProjectTables(user, []string{
		"IUserDeck", "IUserDeckCharacter", "IUserDeckSubWeaponGroup", "IUserDeckPartsGroup",
		"IUserDeckCharacterDressupCostume",
	})
	return &pb.ReplaceTripleDeckResponse{
		DiffUserData: tracker.Apply(user, result),
	}, nil
}
