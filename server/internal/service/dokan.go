package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"
)

type DokanServiceServer struct {
	pb.UnimplementedDokanServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
}

func NewDokanServiceServer(users store.UserRepository, sessions store.SessionRepository) *DokanServiceServer {
	return &DokanServiceServer{users: users, sessions: sessions}
}

func (s *DokanServiceServer) RegisterDokanConfirmed(ctx context.Context, req *pb.RegisterDokanConfirmedRequest) (*pb.RegisterDokanConfirmedResponse, error) {
	log.Printf("[DokanService] RegisterDokanConfirmed: dokanIds=%v", req.DokanId)

	userId := currentUserId(ctx, s.users, s.sessions)
	snapshot, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		for _, id := range req.DokanId {
			user.DokanConfirmed[id] = true
		}
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	diff := userdata.BuildDiffFromTables(userdata.ProjectTables(snapshot, []string{"IUserDokan"}))

	return &pb.RegisterDokanConfirmedResponse{
		DiffUserData: diff,
	}, nil
}
