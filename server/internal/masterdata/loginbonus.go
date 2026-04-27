package masterdata

import (
	"log"
	"lunar-tear/server/internal/utils"
)

type loginBonusStampKey struct {
	LoginBonusId    int32
	LowerPageNumber int32
	StampNumber     int32
}

type LoginBonusReward struct {
	PossessionType int32
	PossessionId   int32
	Count          int32
}

type LoginBonusCatalog struct {
	stamps map[loginBonusStampKey]LoginBonusReward
}

func (c *LoginBonusCatalog) LookupStampReward(loginBonusId, pageNumber, stampNumber int32) (LoginBonusReward, bool) {
	entry, ok := c.stamps[loginBonusStampKey{loginBonusId, pageNumber, stampNumber}]
	return entry, ok
}

func LoadLoginBonusCatalog() *LoginBonusCatalog {
	stamps, err := utils.ReadTable[EntityMLoginBonusStamp]("m_login_bonus_stamp")
	if err != nil {
		log.Fatalf("load login bonus stamp table: %v", err)
	}

	cat := &LoginBonusCatalog{
		stamps: make(map[loginBonusStampKey]LoginBonusReward, len(stamps)),
	}
	for _, s := range stamps {
		cat.stamps[loginBonusStampKey{s.LoginBonusId, s.LowerPageNumber, s.StampNumber}] = LoginBonusReward{
			PossessionType: s.RewardPossessionType,
			PossessionId:   s.RewardPossessionId,
			Count:          s.RewardCount,
		}
	}
	return cat
}
