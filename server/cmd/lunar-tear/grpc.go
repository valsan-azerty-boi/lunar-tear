package main

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/service"
	"lunar-tear/server/internal/store"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type loggingListener struct {
	net.Listener
}

func (l loggingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		log.Printf("[gRPC] Accept error: %v", err)
		return nil, err
	}
	log.Printf("[gRPC] New connection from %v", conn.RemoteAddr())
	return conn, nil
}

func startGRPC(
	host string,
	octoURL string,
	userStore interface {
		store.UserRepository
		store.SessionRepository
	},
	questEngine *questflow.QuestHandler,
	gachaHandler *gacha.GachaHandler,
	gachaEntries []store.GachaCatalogEntry,
	cageOrnamentCatalog *masterdata.CageOrnamentCatalog,
	loginBonusCatalog *masterdata.LoginBonusCatalog,
	characterViewerCatalog *masterdata.CharacterViewerCatalog,
	shopCatalog *masterdata.ShopCatalog,
	costumeCatalog *masterdata.CostumeCatalog,
	omikujiCatalog *masterdata.OmikujiCatalog,
	weaponCatalog *masterdata.WeaponCatalog,
	exploreCatalog *masterdata.ExploreCatalog,
	gimmickCatalog *masterdata.GimmickCatalog,
	characterBoardCatalog *masterdata.CharacterBoardCatalog,
	partsCatalog *masterdata.PartsCatalog,
	characterRebirthCatalog *masterdata.CharacterRebirthCatalog,
	companionCatalog *masterdata.CompanionCatalog,
	materialCatalog *masterdata.MaterialCatalog,
	consumableItemCatalog *masterdata.ConsumableItemCatalog,
	gameConfig *masterdata.GameConfig,
	sideStoryCatalog *masterdata.SideStoryCatalog,
	bigHuntCatalog *masterdata.BigHuntCatalog,
) {
	lis, err := net.Listen("tcp", ":443")
	if err != nil {
		log.Fatalf("failed to listen on :443: %v", err)
	}
	lis = loggingListener{Listener: lis}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(loggingInterceptor, timeSyncInterceptor),
		grpc.UnknownServiceHandler(loggingUnknownService),
	)

	registerServices(grpcServer,
		host,
		octoURL,
		userStore,
		questEngine,
		gachaHandler,
		gachaEntries,
		cageOrnamentCatalog,
		loginBonusCatalog,
		characterViewerCatalog,
		shopCatalog,
		costumeCatalog,
		omikujiCatalog,
		weaponCatalog,
		exploreCatalog,
		gimmickCatalog,
		characterBoardCatalog,
		partsCatalog,
		characterRebirthCatalog,
		companionCatalog,
		materialCatalog,
		consumableItemCatalog,
		gameConfig,
		sideStoryCatalog,
		bigHuntCatalog,
	)

	reflection.Register(grpcServer)

	log.Printf("gRPC server listening on :443")
	log.Printf("client host address: %s:443", host)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func registerServices(
	srv *grpc.Server,
	host string,
	octoURL string,
	userStore interface {
		store.UserRepository
		store.SessionRepository
	},
	questEngine *questflow.QuestHandler,
	gachaHandler *gacha.GachaHandler,
	gachaEntries []store.GachaCatalogEntry,
	cageOrnamentCatalog *masterdata.CageOrnamentCatalog,
	loginBonusCatalog *masterdata.LoginBonusCatalog,
	characterViewerCatalog *masterdata.CharacterViewerCatalog,
	shopCatalog *masterdata.ShopCatalog,
	costumeCatalog *masterdata.CostumeCatalog,
	omikujiCatalog *masterdata.OmikujiCatalog,
	weaponCatalog *masterdata.WeaponCatalog,
	exploreCatalog *masterdata.ExploreCatalog,
	gimmickCatalog *masterdata.GimmickCatalog,
	characterBoardCatalog *masterdata.CharacterBoardCatalog,
	partsCatalog *masterdata.PartsCatalog,
	characterRebirthCatalog *masterdata.CharacterRebirthCatalog,
	companionCatalog *masterdata.CompanionCatalog,
	materialCatalog *masterdata.MaterialCatalog,
	consumableItemCatalog *masterdata.ConsumableItemCatalog,
	gameConfig *masterdata.GameConfig,
	sideStoryCatalog *masterdata.SideStoryCatalog,
	bigHuntCatalog *masterdata.BigHuntCatalog,
) {
	pb.RegisterBannerServiceServer(srv, service.NewBannerServiceServer(gachaEntries))
	pb.RegisterUserServiceServer(srv, service.NewUserServiceServer(userStore, userStore))
	pb.RegisterBattleServiceServer(srv, service.NewBattleServiceServer(userStore, userStore))
	pb.RegisterConfigServiceServer(srv, service.NewConfigServiceServer(host, int32(443), octoURL))
	pb.RegisterDataServiceServer(srv, service.NewDataServiceServer(userStore, userStore))
	pb.RegisterTutorialServiceServer(srv, service.NewTutorialServiceServer(userStore, userStore, questEngine))
	pb.RegisterGachaServiceServer(srv, service.NewGachaServiceServer(userStore, userStore, gachaEntries, gachaHandler))
	pb.RegisterGiftServiceServer(srv, service.NewGiftServiceServer(userStore, userStore))
	pb.RegisterGamePlayServiceServer(srv, service.NewGameplayServiceServer())
	pb.RegisterGimmickServiceServer(srv, service.NewGimmickServiceServer(userStore, userStore, gimmickCatalog))
	pb.RegisterQuestServiceServer(srv, service.NewQuestServiceServer(userStore, userStore, questEngine))
	pb.RegisterNotificationServiceServer(srv, service.NewNotificationServiceServer(userStore, userStore))
	pb.RegisterCageOrnamentServiceServer(srv, service.NewCageOrnamentServiceServer(userStore, userStore, cageOrnamentCatalog, questEngine.Granter))
	pb.RegisterDeckServiceServer(srv, service.NewDeckServiceServer(userStore, userStore))
	pb.RegisterFriendServiceServer(srv, service.NewFriendServiceServer(userStore, userStore))
	pb.RegisterLoginBonusServiceServer(srv, service.NewLoginBonusServiceServer(userStore, userStore, loginBonusCatalog))
	pb.RegisterNaviCutInServiceServer(srv, service.NewNaviCutInServiceServer(userStore, userStore))
	pb.RegisterContentsStoryServiceServer(srv, service.NewContentsStoryServiceServer(userStore, userStore))
	pb.RegisterDokanServiceServer(srv, service.NewDokanServiceServer(userStore, userStore))
	pb.RegisterPortalCageServiceServer(srv, service.NewPortalCageServiceServer(userStore, userStore))
	pb.RegisterCharacterViewerServiceServer(srv, service.NewCharacterViewerServiceServer(userStore, userStore, characterViewerCatalog))
	pb.RegisterMissionServiceServer(srv, service.NewMissionServiceServer(userStore, userStore))
	pb.RegisterShopServiceServer(srv, service.NewShopServiceServer(userStore, userStore, shopCatalog, questEngine.Granter))
	pb.RegisterCostumeServiceServer(srv, service.NewCostumeServiceServer(userStore, userStore, costumeCatalog, gameConfig))
	pb.RegisterMovieServiceServer(srv, service.NewMovieServiceServer(userStore, userStore))
	pb.RegisterOmikujiServiceServer(srv, service.NewOmikujiServiceServer(userStore, userStore, omikujiCatalog))
	pb.RegisterWeaponServiceServer(srv, service.NewWeaponServiceServer(userStore, userStore, weaponCatalog, gameConfig))
	pb.RegisterExploreServiceServer(srv, service.NewExploreServiceServer(userStore, userStore, exploreCatalog))
	pb.RegisterCharacterBoardServiceServer(srv, service.NewCharacterBoardServiceServer(userStore, userStore, characterBoardCatalog))
	pb.RegisterPartsServiceServer(srv, service.NewPartsServiceServer(userStore, userStore, partsCatalog, gameConfig))
	pb.RegisterCharacterServiceServer(srv, service.NewCharacterServiceServer(userStore, userStore, characterRebirthCatalog, gameConfig))
	pb.RegisterCompanionServiceServer(srv, service.NewCompanionServiceServer(userStore, userStore, companionCatalog, gameConfig))
	pb.RegisterMaterialServiceServer(srv, service.NewMaterialServiceServer(userStore, userStore, materialCatalog, gameConfig))
	pb.RegisterConsumableItemServiceServer(srv, service.NewConsumableItemServiceServer(userStore, userStore, consumableItemCatalog, gameConfig))
	pb.RegisterSideStoryQuestServiceServer(srv, service.NewSideStoryQuestServiceServer(userStore, userStore, sideStoryCatalog))
	pb.RegisterBigHuntServiceServer(srv, service.NewBigHuntServiceServer(userStore, userStore, bigHuntCatalog, questEngine))
	pb.RegisterRewardServiceServer(srv, service.NewRewardServiceServer(userStore, userStore, bigHuntCatalog, questEngine.Granter))
}

func loggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	log.Printf(">>> %s", info.FullMethod)
	resp, err := handler(ctx, req)
	if err != nil {
		log.Printf("<<< %s ERROR: %v", info.FullMethod, err)
	} else {
		log.Printf("<<< %s OK", info.FullMethod)
	}
	return resp, err
}

func timeSyncInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	resp, err := handler(ctx, req)
	switch info.FullMethod {
	case "/apb.api.user.UserService/Auth",
		"/apb.api.user.UserService/RegisterUser",
		"/apb.api.user.UserService/TransferUser":
	default:
		grpc.SetTrailer(ctx, metadata.Pairs(
			"x-apb-response-datetime", fmt.Sprintf("%d", gametime.NowMillis()),
		))
	}
	return resp, err
}

func loggingUnknownService(_ any, stream grpc.ServerStream) error {
	fullMethod, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		fullMethod = "<unknown>"
	}
	log.Printf(">>> %s", fullMethod)
	err := status.Errorf(codes.Unimplemented, "unknown service or method %s", fullMethod)
	log.Printf("<<< %s ERROR: %v", fullMethod, err)
	return err
}
