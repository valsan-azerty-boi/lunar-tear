package service

import (
	"context"
	"fmt"
	"log"
	"os"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/schedule"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"

	"google.golang.org/protobuf/types/known/emptypb"
)

type DataServiceServer struct {
	pb.UnimplementedDataServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	schedule *schedule.Manager
}

func NewDataServiceServer(users store.UserRepository, sessions store.SessionRepository, schedule *schedule.Manager) *DataServiceServer {
	return &DataServiceServer{users: users, sessions: sessions, schedule: schedule}
}

func (s *DataServiceServer) GetLatestMasterDataVersion(ctx context.Context, _ *emptypb.Empty) (*pb.MasterDataGetLatestVersionResponse, error) {
	versionStamp := int64(0)
	if info, err := os.Stat("assets/release/database.bin.e"); err == nil {
		versionStamp = info.ModTime().UnixMilli()
	}

	version := fmt.Sprintf("20240404193219_%d", versionStamp)
	log.Printf("[DataService] GetLatestMasterDataVersion -> %s", version)
	return &pb.MasterDataGetLatestVersionResponse{
		LatestMasterDataVersion: version,
	}, nil
}

func (s *DataServiceServer) GetUserDataNameV2(ctx context.Context, _ *emptypb.Empty) (*pb.UserDataGetNameResponseV2, error) {
	log.Printf("[DataService] GetUserDataNameV2")
	return &pb.UserDataGetNameResponseV2{
		TableNameList: []*pb.TableNameList{
			{TableName: defaultTableNames()},
		},
	}, nil
}

func (s *DataServiceServer) GetUserData(ctx context.Context, req *pb.UserDataGetRequest) (*pb.UserDataGetResponse, error) {
	log.Printf("[DataService] GetUserData: tables=%v", req.TableName)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return nil, fmt.Errorf("snapshot user: %w", err)
	}

	defaults := userdata.FullClientTableMap(user)
	result := userdata.SelectTables(defaults, req.TableName)
	return &pb.UserDataGetResponse{
		UserDataJson: result,
	}, nil
}

func defaultTableNames() []string {
	return []string{
		"IUser",
		"IUserApple",
		"IUserAutoSaleSettingDetail",
		"IUserBeginnerCampaign",
		"IUserBigHuntMaxScore",
		"IUserBigHuntProgressStatus",
		"IUserBigHuntScheduleMaxScore",
		"IUserBigHuntStatus",
		"IUserBigHuntWeeklyMaxScore",
		"IUserBigHuntWeeklyStatus",
		"IUserCageOrnamentReward",
		"IUserCharacter",
		"IUserCharacterBoard",
		"IUserCharacterBoardAbility",
		"IUserCharacterBoardCompleteReward",
		"IUserCharacterBoardStatusUp",
		"IUserCharacterCostumeLevelBonus",
		"IUserCharacterRebirth",
		"IUserCharacterViewerField",
		"IUserComebackCampaign",
		"IUserCompanion",
		"IUserConsumableItem",
		"IUserContentsStory",
		"IUserCostume",
		"IUserCostumeActiveSkill",
		"IUserCostumeAwakenStatusUp",
		"IUserCostumeLevelBonusReleaseStatus",
		"IUserCostumeLotteryEffect",
		"IUserCostumeLotteryEffectAbility",
		"IUserCostumeLotteryEffectPending",
		"IUserCostumeLotteryEffectStatusUp",
		"IUserDeck",
		"IUserDeckCharacter",
		"IUserDeckCharacterDressupCostume",
		"IUserDeckLimitContentRestricted",
		"IUserDeckPartsGroup",
		"IUserDeckSubWeaponGroup",
		"IUserDeckTypeNote",
		"IUserDokan",
		"IUserEventQuestDailyGroupCompleteReward",
		"IUserEventQuestGuerrillaFreeOpen",
		"IUserEventQuestLabyrinthSeason",
		"IUserEventQuestLabyrinthStage",
		"IUserEventQuestProgressStatus",
		"IUserEventQuestTowerAccumulationReward",
		"IUserExplore",
		"IUserExploreScore",
		"IUserExtraQuestProgressStatus",
		"IUserFacebook",
		"IUserGem",
		"IUserGimmick",
		"IUserGimmickOrnamentProgress",
		"IUserGimmickSequence",
		"IUserGimmickUnlock",
		"IUserImportantItem",
		"IUserLimitedOpen",
		// "IUserLogin",
		"IUserLoginBonus",
		"IUserMainQuestFlowStatus",
		"IUserMainQuestMainFlowStatus",
		"IUserMainQuestProgressStatus",
		"IUserMainQuestReplayFlowStatus",
		"IUserMainQuestSeasonRoute",
		"IUserMaterial",
		"IUserMission",
		"IUserMissionCompletionProgress",
		"IUserMissionPassPoint",
		"IUserMovie",
		"IUserNaviCutIn",
		"IUserOmikuji",
		"IUserParts",
		"IUserPartsGroupNote",
		"IUserPartsPreset",
		"IUserPartsPresetTag",
		"IUserPartsStatusSub",
		"IUserPortalCageStatus",
		"IUserPossessionAutoConvert",
		"IUserPremiumItem",
		"IUserProfile",
		"IUserPvpDefenseDeck",
		"IUserPvpStatus",
		"IUserPvpWeeklyResult",
		"IUserQuest",
		"IUserQuestAutoOrbit",
		"IUserQuestLimitContentStatus",
		"IUserQuestMission",
		"IUserQuestReplayFlowRewardGroup",
		"IUserQuestSceneChoice",
		"IUserQuestSceneChoiceHistory",
		// "IUserSetting",
		"IUserShopItem",
		"IUserShopReplaceable",
		"IUserShopReplaceableLineup",
		"IUserSideStoryQuest",
		"IUserSideStoryQuestSceneProgressStatus",
		"IUserStatus",
		"IUserThought",
		"IUserTripleDeck",
		"IUserTutorialProgress",
		"IUserWeapon",
		"IUserWeaponAbility",
		"IUserWeaponAwaken",
		"IUserWeaponNote",
		"IUserWeaponSkill",
		"IUserWeaponStory",
		"IUserWebviewPanelMission",
	}
}
