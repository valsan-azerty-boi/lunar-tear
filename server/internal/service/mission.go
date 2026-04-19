package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"
)

type MissionServiceServer struct {
	pb.UnimplementedMissionServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
}

func NewMissionServiceServer(users store.UserRepository, sessions store.SessionRepository) *MissionServiceServer {
	return &MissionServiceServer{users: users, sessions: sessions}
}

func (s *MissionServiceServer) UpdateMissionProgress(ctx context.Context, req *pb.UpdateMissionProgressRequest) (*pb.UpdateMissionProgressResponse, error) {
	log.Printf("[MissionService] UpdateMissionProgress: cage=%v pictureBook=%v", req.CageMeasurableValues, req.PictureBookMeasurableValues)

	userId := currentUserId(ctx, s.users, s.sessions)
	snapshot, err := s.users.LoadUser(userId)
	if err != nil {
		return nil, fmt.Errorf("snapshot user: %w", err)
	}

	diff := userdata.BuildDiffFromTables(userdata.ProjectTables(snapshot, []string{"IUserMission"}))

	return &pb.UpdateMissionProgressResponse{
		DiffUserData: diff,
	}, nil
}
