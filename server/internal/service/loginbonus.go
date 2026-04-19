package service

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type LoginBonusServiceServer struct {
	pb.UnimplementedLoginBonusServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.LoginBonusCatalog
}

func NewLoginBonusServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.LoginBonusCatalog) *LoginBonusServiceServer {
	return &LoginBonusServiceServer{users: users, sessions: sessions, catalog: catalog}
}

func (s *LoginBonusServiceServer) ReceiveStamp(ctx context.Context, req *emptypb.Empty) (*pb.ReceiveStampResponse, error) {
	log.Printf("[LoginBonusService] ReceiveStamp")
	userId := currentUserId(ctx, s.users, s.sessions)

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		now := gametime.NowMillis()
		nextStamp := user.LoginBonus.CurrentStampNumber + 1

		reward, ok := s.catalog.LookupStampReward(
			user.LoginBonus.LoginBonusId,
			user.LoginBonus.CurrentPageNumber,
			nextStamp,
		)
		if !ok {
			log.Fatalf("[LoginBonusService] no reward found for bonusId=%d page=%d stamp=%d",
				user.LoginBonus.LoginBonusId, user.LoginBonus.CurrentPageNumber, nextStamp)
		}

		log.Printf("[LoginBonusService] stamp %d -> possType=%d possId=%d count=%d (-> gift box)",
			nextStamp, reward.PossessionType, reward.PossessionId, reward.Count)

		user.Gifts.NotReceived = append(user.Gifts.NotReceived, store.NotReceivedGiftState{
			GiftCommon: store.GiftCommonState{
				PossessionType: reward.PossessionType,
				PossessionId:   reward.PossessionId,
				Count:          reward.Count,
				GrantDatetime:  now,
			},
			ExpirationDatetime: now + int64(30*24*time.Hour/time.Millisecond),
			UserGiftUuid:       uuid.New().String(),
		})
		user.Notifications.GiftNotReceiveCount = int32(len(user.Gifts.NotReceived))
		user.LoginBonus.CurrentStampNumber = nextStamp
		user.LoginBonus.LatestRewardReceiveDatetime = now
		user.LoginBonus.LatestVersion = now
	})

	diff := userdata.BuildDiffFromTables(userdata.ProjectTables(user,
		[]string{"IUserLoginBonus"},
	))
	setCommonResponseTrailers(ctx, diff, false)
	return &pb.ReceiveStampResponse{DiffUserData: diff}, nil
}
