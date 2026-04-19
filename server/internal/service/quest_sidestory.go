package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"
)

type SideStoryQuestServiceServer struct {
	pb.UnimplementedSideStoryQuestServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.SideStoryCatalog
}

func NewSideStoryQuestServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.SideStoryCatalog) *SideStoryQuestServiceServer {
	return &SideStoryQuestServiceServer{users: users, sessions: sessions, catalog: catalog}
}

func buildSideStoryDiff(user store.UserState, tableNames []string) map[string]*pb.DiffData {
	tables := userdata.ProjectTables(user, tableNames)
	return userdata.BuildDiffFromTablesOrdered(tables, tableNames)
}

func (s *SideStoryQuestServiceServer) MoveSideStoryQuestProgress(ctx context.Context, req *pb.MoveSideStoryQuestRequest) (*pb.MoveSideStoryQuestResponse, error) {
	log.Printf("[SideStoryQuestService] MoveSideStoryQuestProgress: sideStoryQuestId=%d", req.SideStoryQuestId)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	firstSceneId := s.catalog.FirstSceneByQuestId[req.SideStoryQuestId]

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		existing, exists := user.SideStoryQuests[req.SideStoryQuestId]

		var sceneId int32
		if exists && existing.HeadSideStoryQuestSceneId > 0 {
			sceneId = existing.HeadSideStoryQuestSceneId
		} else {
			sceneId = firstSceneId
		}

		user.SideStoryActiveProgress.CurrentSideStoryQuestId = req.SideStoryQuestId
		user.SideStoryActiveProgress.CurrentSideStoryQuestSceneId = sceneId
		user.SideStoryActiveProgress.LatestVersion = nowMillis

		if !exists {
			user.SideStoryQuests[req.SideStoryQuestId] = store.SideStoryQuestProgress{
				HeadSideStoryQuestSceneId: firstSceneId,
				SideStoryQuestStateType:   model.SideStoryQuestStateActive,
				LatestVersion:             nowMillis,
			}
		}
	})

	return &pb.MoveSideStoryQuestResponse{
		DiffUserData: buildSideStoryDiff(user, []string{
			"IUserSideStoryQuest",
			"IUserSideStoryQuestSceneProgressStatus",
		}),
	}, nil
}

func (s *SideStoryQuestServiceServer) UpdateSideStoryQuestSceneProgress(ctx context.Context, req *pb.UpdateSideStoryQuestSceneProgressRequest) (*pb.UpdateSideStoryQuestSceneProgressResponse, error) {
	log.Printf("[SideStoryQuestService] UpdateSideStoryQuestSceneProgress: sideStoryQuestId=%d sceneId=%d",
		req.SideStoryQuestId, req.SideStoryQuestSceneId)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		user.SideStoryActiveProgress.CurrentSideStoryQuestSceneId = req.SideStoryQuestSceneId
		user.SideStoryActiveProgress.LatestVersion = nowMillis

		progress := user.SideStoryQuests[req.SideStoryQuestId]
		if req.SideStoryQuestSceneId > progress.HeadSideStoryQuestSceneId {
			progress.HeadSideStoryQuestSceneId = req.SideStoryQuestSceneId
		}
		progress.LatestVersion = nowMillis
		user.SideStoryQuests[req.SideStoryQuestId] = progress
	})

	return &pb.UpdateSideStoryQuestSceneProgressResponse{
		DiffUserData: buildSideStoryDiff(user, []string{
			"IUserSideStoryQuest",
			"IUserSideStoryQuestSceneProgressStatus",
		}),
	}, nil
}
