package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"
)

type PortalCageServiceServer struct {
	pb.UnimplementedPortalCageServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
}

func NewPortalCageServiceServer(users store.UserRepository, sessions store.SessionRepository) *PortalCageServiceServer {
	return &PortalCageServiceServer{users: users, sessions: sessions}
}

func (s *PortalCageServiceServer) UpdatePortalCageSceneProgress(ctx context.Context, req *pb.UpdatePortalCageSceneProgressRequest) (*pb.UpdatePortalCageSceneProgressResponse, error) {
	log.Printf("[PortalCageService] UpdatePortalCageSceneProgress: portalCageSceneId=%d", req.PortalCageSceneId)

	userId := currentUserId(ctx, s.users, s.sessions)
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		now := gametime.NowMillis()
		user.PortalCageStatus.IsCurrentProgress = true
		user.PortalCageStatus.LatestVersion = now
	})

	tables := userdata.ProjectTables(user,
		[]string{"IUserPortalCageStatus"},
	)
	return &pb.UpdatePortalCageSceneProgressResponse{
		DiffUserData: userdata.BuildDiffFromTablesOrdered(tables, []string{"IUserPortalCageStatus"}),
	}, nil
}
