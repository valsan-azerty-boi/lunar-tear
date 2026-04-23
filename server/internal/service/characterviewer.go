package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/store"

	"google.golang.org/protobuf/types/known/emptypb"
)

type CharacterViewerServiceServer struct {
	pb.UnimplementedCharacterViewerServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.CharacterViewerCatalog
}

func NewCharacterViewerServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.CharacterViewerCatalog) *CharacterViewerServiceServer {
	return &CharacterViewerServiceServer{users: users, sessions: sessions, catalog: catalog}
}

func (s *CharacterViewerServiceServer) CharacterViewerTop(ctx context.Context, _ *emptypb.Empty) (*pb.CharacterViewerTopResponse, error) {
	log.Printf("[CharacterViewerService] CharacterViewerTop")

	userId := CurrentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		panic(fmt.Sprintf("CharacterViewerTop: no user for userId=%d: %v", userId, err))
	}

	released := s.catalog.ReleasedFieldIds(user)
	log.Printf("[CharacterViewerService] released %d fields for user %d", len(released), userId)

	return &pb.CharacterViewerTopResponse{
		ReleaseCharacterViewerFieldId: released,
	}, nil
}
