package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
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

	userId := currentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		panic(fmt.Sprintf("CharacterViewerTop: no user for userId=%d: %v", userId, err))
	}

	released := s.catalog.ReleasedFieldIds(user)
	log.Printf("[CharacterViewerService] released %d fields for user %d", len(released), userId)

	now := gametime.NowMillis()
	records := make([]map[string]any, 0, len(released))
	for _, fieldId := range released {
		records = append(records, map[string]any{
			"userId":                 userId,
			"characterViewerFieldId": fieldId,
			"releaseDatetime":        now,
			"latestVersion":          0,
		})
	}

	payload := "[]"
	if len(records) > 0 {
		data, _ := json.Marshal(records)
		payload = string(data)
	}

	diff := map[string]*pb.DiffData{
		"IUserCharacterViewerField": {
			UpdateRecordsJson: payload,
			DeleteKeysJson:    "[]",
		},
	}

	return &pb.CharacterViewerTopResponse{
		ReleaseCharacterViewerFieldId: released,
		DiffUserData:                  diff,
	}, nil
}
