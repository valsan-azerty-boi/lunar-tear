package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"
)

type ContentsStoryServiceServer struct {
	pb.UnimplementedContentsStoryServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
}

func NewContentsStoryServiceServer(users store.UserRepository, sessions store.SessionRepository) *ContentsStoryServiceServer {
	return &ContentsStoryServiceServer{users: users, sessions: sessions}
}

func (s *ContentsStoryServiceServer) RegisterPlayed(ctx context.Context, req *pb.ContentsStoryRegisterPlayedRequest) (*pb.ContentsStoryRegisterPlayedResponse, error) {
	log.Printf("[ContentsStoryService] RegisterPlayed: contentsStoryId=%d", req.ContentsStoryId)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	snapshot, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		user.ContentsStories[req.ContentsStoryId] = nowMillis
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	diff := userdata.BuildDiffFromTables(userdata.ProjectTables(snapshot, []string{"IUserContentsStory"}))

	return &pb.ContentsStoryRegisterPlayedResponse{
		DiffUserData: diff,
	}, nil
}
