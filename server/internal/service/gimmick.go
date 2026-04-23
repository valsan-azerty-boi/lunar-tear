package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/store"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type GimmickServiceServer struct {
	pb.UnimplementedGimmickServiceServer
	users          store.UserRepository
	sessions       store.SessionRepository
	gimmickCatalog *masterdata.GimmickCatalog
}

func NewGimmickServiceServer(users store.UserRepository, sessions store.SessionRepository, gimmickCatalog *masterdata.GimmickCatalog) *GimmickServiceServer {
	return &GimmickServiceServer{users: users, sessions: sessions, gimmickCatalog: gimmickCatalog}
}

func (s *GimmickServiceServer) UpdateSequence(ctx context.Context, req *pb.UpdateSequenceRequest) (*pb.UpdateSequenceResponse, error) {
	log.Printf("[GimmickService] UpdateSequence: scheduleId=%d sequenceId=%d",
		req.GimmickSequenceScheduleId, req.GimmickSequenceId)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		key := store.GimmickSequenceKey{
			GimmickSequenceScheduleId: req.GimmickSequenceScheduleId,
			GimmickSequenceId:         req.GimmickSequenceId,
		}
		sequence := user.Gimmick.Sequences[key]
		sequence.Key = key
		user.Gimmick.Sequences[key] = sequence
	})
	return &pb.UpdateSequenceResponse{}, nil
}

func (s *GimmickServiceServer) UpdateGimmickProgress(ctx context.Context, req *pb.UpdateGimmickProgressRequest) (*pb.UpdateGimmickProgressResponse, error) {
	log.Printf("[GimmickService] UpdateGimmickProgress: scheduleId=%d sequenceId=%d gimmickId=%d ornamentIndex=%d progressValueBit=%d flowType=%d",
		req.GimmickSequenceScheduleId, req.GimmickSequenceId, req.GimmickId, req.GimmickOrnamentIndex, req.ProgressValueBit, req.FlowType)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		nowMillis := gametime.NowMillis()
		progressKey := store.GimmickKey{
			GimmickSequenceScheduleId: req.GimmickSequenceScheduleId,
			GimmickSequenceId:         req.GimmickSequenceId,
			GimmickId:                 req.GimmickId,
		}
		progress := user.Gimmick.Progress[progressKey]
		progress.Key = progressKey
		progress.StartDatetime = nowMillis
		user.Gimmick.Progress[progressKey] = progress

		ornamentKey := store.GimmickOrnamentKey{
			GimmickSequenceScheduleId: req.GimmickSequenceScheduleId,
			GimmickSequenceId:         req.GimmickSequenceId,
			GimmickId:                 req.GimmickId,
			GimmickOrnamentIndex:      req.GimmickOrnamentIndex,
		}
		ornament := user.Gimmick.OrnamentProgress[ornamentKey]
		ornament.Key = ornamentKey
		ornament.ProgressValueBit = req.ProgressValueBit
		ornament.BaseDatetime = nowMillis
		user.Gimmick.OrnamentProgress[ornamentKey] = ornament
	})
	return &pb.UpdateGimmickProgressResponse{
		GimmickOrnamentReward:      []*pb.GimmickReward{},
		IsSequenceCleared:          false,
		GimmickSequenceClearReward: []*pb.GimmickReward{},
	}, nil
}

func (s *GimmickServiceServer) InitSequenceSchedule(ctx context.Context, _ *emptypb.Empty) (*pb.InitSequenceScheduleResponse, error) {
	log.Printf("[GimmickService] InitSequenceSchedule")
	userId := CurrentUserId(ctx, s.users, s.sessions)
	now := gametime.NowMillis()
	s.users.UpdateUser(userId, func(user *store.UserState) {
		added := 0
		for _, key := range s.gimmickCatalog.ActiveScheduleKeys(*user, now) {
			if _, exists := user.Gimmick.Sequences[key]; !exists {
				user.Gimmick.Sequences[key] = store.GimmickSequenceState{Key: key}
				added++
			}
		}
		if added > 0 {
			log.Printf("[GimmickService] InitSequenceSchedule: added %d sequences (total %d)", added, len(user.Gimmick.Sequences))
		}
	})
	return &pb.InitSequenceScheduleResponse{}, nil
}

func (s *GimmickServiceServer) Unlock(ctx context.Context, req *pb.UnlockRequest) (*pb.UnlockResponse, error) {
	log.Printf("[GimmickService] Unlock: gimmickKeys=%d", len(req.GimmickKey))
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		for _, item := range req.GimmickKey {
			key := store.GimmickKey{
				GimmickSequenceScheduleId: item.GimmickSequenceScheduleId,
				GimmickSequenceId:         item.GimmickSequenceId,
				GimmickId:                 item.GimmickId,
			}
			unlock := user.Gimmick.Unlocks[key]
			unlock.Key = key
			unlock.IsUnlocked = true
			user.Gimmick.Unlocks[key] = unlock
		}
	})
	return &pb.UnlockResponse{}, nil
}
