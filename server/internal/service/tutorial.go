package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
)

type TutorialServiceServer struct {
	pb.UnimplementedTutorialServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	engine   *questflow.QuestHandler
}

func NewTutorialServiceServer(users store.UserRepository, sessions store.SessionRepository, engine *questflow.QuestHandler) *TutorialServiceServer {
	return &TutorialServiceServer{users: users, sessions: sessions, engine: engine}
}

func (s *TutorialServiceServer) SetTutorialProgress(ctx context.Context, req *pb.SetTutorialProgressRequest) (*pb.SetTutorialProgressResponse, error) {
	log.Printf("[TutorialService] SetTutorialProgress: type=%d phase=%d choice=%d", req.TutorialType, req.ProgressPhase, req.ChoiceId)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	var grants []questflow.RewardGrant
	s.users.UpdateUser(userId, func(user *store.UserState) {
		existing, exists := user.Tutorials[req.TutorialType]
		if !exists || req.ProgressPhase >= existing.ProgressPhase {
			user.Tutorials[req.TutorialType] = store.TutorialProgressState{
				TutorialType:  req.TutorialType,
				ProgressPhase: req.ProgressPhase,
				ChoiceId:      req.ChoiceId,
			}
		}
		grants = s.engine.ApplyTutorialReward(user, model.TutorialType(req.TutorialType), req.ChoiceId, nowMillis)
		if req.TutorialType == int32(model.TutorialTypeMenuFirst) && req.ProgressPhase == 20 {
			store.EnsureDefaultDeck(user, nowMillis)
		}
	})

	rewards := make([]*pb.TutorialChoiceReward, len(grants))
	for i, g := range grants {
		rewards[i] = &pb.TutorialChoiceReward{
			PossessionType: int32(g.PossessionType),
			PossessionId:   g.PossessionId,
			Count:          g.Count,
		}
	}
	return &pb.SetTutorialProgressResponse{
		TutorialChoiceReward: rewards,
	}, nil
}

func (s *TutorialServiceServer) SetTutorialProgressAndReplaceDeck(ctx context.Context, req *pb.SetTutorialProgressAndReplaceDeckRequest) (*pb.SetTutorialProgressAndReplaceDeckResponse, error) {
	log.Printf("[TutorialService] SetTutorialProgressAndReplaceDeck: type=%d phase=%d deckType=%d deckNumber=%d", req.TutorialType, req.ProgressPhase, req.DeckType, req.UserDeckNumber)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		existing, exists := user.Tutorials[req.TutorialType]
		if !exists || req.ProgressPhase >= existing.ProgressPhase {
			user.Tutorials[req.TutorialType] = store.TutorialProgressState{
				TutorialType:  req.TutorialType,
				ProgressPhase: req.ProgressPhase,
			}
		}
		if req.Deck != nil {
			store.ApplyDeckReplacement(user, model.DeckType(req.DeckType), req.UserDeckNumber, deckSlotsFromProto(req.Deck), gametime.NowMillis())
		}
	})
	return &pb.SetTutorialProgressAndReplaceDeckResponse{}, nil
}
