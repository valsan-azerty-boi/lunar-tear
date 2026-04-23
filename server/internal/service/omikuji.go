package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/store"
)

type OmikujiServiceServer struct {
	pb.UnimplementedOmikujiServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.OmikujiCatalog
}

func NewOmikujiServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.OmikujiCatalog) *OmikujiServiceServer {
	return &OmikujiServiceServer{users: users, sessions: sessions, catalog: catalog}
}

func (s *OmikujiServiceServer) OmikujiDraw(ctx context.Context, req *pb.OmikujiDrawRequest) (*pb.OmikujiDrawResponse, error) {
	log.Printf("[OmikujiService] OmikujiDraw: omikujiId=%d", req.OmikujiId)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	now := gametime.NowMillis()

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		user.DrawnOmikuji[req.OmikujiId] = now
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return &pb.OmikujiDrawResponse{
		OmikujiResultAssetId: s.catalog.LookupAssetId(req.OmikujiId),
		OmikujiItem:          []*pb.OmikujiItem{},
	}, nil
}
