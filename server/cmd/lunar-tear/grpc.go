package main

import (
	"log"
	"net"
	"strconv"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/interceptor"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/service"
	"lunar-tear/server/internal/store"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	listenAddr string,
	publicAddr string,
	octoURL string,
	authURL string,
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
) *grpc.Server {
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", listenAddr, err)
	}
	lis = loggingListener{Listener: lis}

	diffInterceptor := interceptor.NewDiffInterceptor(userStore, userStore)
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptor.Platform, interceptor.Logging, diffInterceptor, interceptor.TimeSync),
		grpc.UnknownServiceHandler(interceptor.UnknownService),
	)

	registerServices(grpcServer,
		publicAddr,
		octoURL,
		authURL,
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

	log.Printf("gRPC server listening on %s", lis.Addr())
	log.Printf("public address: %s", publicAddr)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()
	return grpcServer
}

func registerServices(
	srv *grpc.Server,
	publicAddr string,
	octoURL string,
	authURL string,
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
	pubHost, pubPortStr, _ := net.SplitHostPort(publicAddr)
	pubPort, _ := strconv.Atoi(pubPortStr)

	pb.RegisterBannerServiceServer(srv, service.NewBannerServiceServer(gachaEntries))
	pb.RegisterUserServiceServer(srv, service.NewUserServiceServer(userStore, userStore, authURL))
	pb.RegisterBattleServiceServer(srv, service.NewBattleServiceServer(userStore, userStore))
	pb.RegisterConfigServiceServer(srv, service.NewConfigServiceServer(pubHost, int32(pubPort), octoURL))
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
