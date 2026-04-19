package store

import (
	"fmt"
	"log"
	"sort"

	"github.com/google/uuid"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
)

func DeductPrice(user *UserState, priceType, priceId, amount int32) error {
	switch priceType {
	case model.PriceTypeConsumableItem:
		cur := user.ConsumableItems[priceId]
		if cur < amount {
			return fmt.Errorf("insufficient consumable %d: have %d, need %d", priceId, cur, amount)
		}
		user.ConsumableItems[priceId] = cur - amount
	case model.PriceTypeGem:
		total := user.Gem.FreeGem + user.Gem.PaidGem
		if total < amount {
			return fmt.Errorf("insufficient gems: have %d, need %d", total, amount)
		}
		if user.Gem.FreeGem >= amount {
			user.Gem.FreeGem -= amount
		} else {
			amount -= user.Gem.FreeGem
			user.Gem.FreeGem = 0
			user.Gem.PaidGem -= amount
		}
	case model.PriceTypePaidGem:
		if user.Gem.PaidGem < amount {
			return fmt.Errorf("insufficient paid gems: have %d, need %d", user.Gem.PaidGem, amount)
		}
		user.Gem.PaidGem -= amount
	case model.PriceTypePlatformPayment:
		// real-money purchase -- treat as free on private server
	default:
		log.Printf("[DeductPrice] unhandled priceType=%d priceId=%d amount=%d", priceType, priceId, amount)
	}
	return nil
}

func DeductPossession(user *UserState, possessionType model.PossessionType, possessionId, count int32) {
	switch possessionType {
	case model.PossessionTypeMaterial:
		user.Materials[possessionId] -= count
		if user.Materials[possessionId] <= 0 {
			delete(user.Materials, possessionId)
		}
	case model.PossessionTypeConsumableItem:
		user.ConsumableItems[possessionId] -= count
		if user.ConsumableItems[possessionId] <= 0 {
			delete(user.ConsumableItems, possessionId)
		}
	case model.PossessionTypePaidGem:
		user.Gem.PaidGem -= count
	case model.PossessionTypeFreeGem:
		user.Gem.FreeGem -= count
	default:
		log.Printf("[DeductPossession] unhandled type=%d id=%d count=%d", possessionType, possessionId, count)
	}
}

func GrantPossession(user *UserState, possessionType model.PossessionType, possessionId, count int32) {
	switch possessionType {
	case model.PossessionTypeMaterial:
		user.Materials[possessionId] += count
	case model.PossessionTypeConsumableItem:
		user.ConsumableItems[possessionId] += count
	case model.PossessionTypePaidGem:
		user.Gem.PaidGem += count
	case model.PossessionTypeFreeGem:
		user.Gem.FreeGem += count
	case model.PossessionTypeImportantItem:
		user.ImportantItems[possessionId] += count
	case model.PossessionTypePremiumItem:
		user.PremiumItems[possessionId] = gametime.NowMillis()
	default:
		log.Printf("[GrantPossession] unhandled type=%d id=%d count=%d", possessionType, possessionId, count)
	}
}

type CostumeRef struct {
	CharacterId int32
}

type WeaponRef struct {
	WeaponSkillGroupId                 int32
	WeaponAbilityGroupId               int32
	WeaponStoryReleaseConditionGroupId int32
}

type WeaponStoryReleaseCond struct {
	StoryIndex                      int32
	WeaponStoryReleaseConditionType model.WeaponStoryReleaseConditionType
	ConditionValue                  int32
}

type PossessionGranter struct {
	CostumeById        map[int32]CostumeRef
	WeaponById         map[int32]WeaponRef
	WeaponSkillSlots   map[int32][]int32
	WeaponAbilitySlots map[int32][]int32
	ReleaseConditions  map[int32][]WeaponStoryReleaseCond

	LastChangedStoryWeaponIds []int32
}

func (g *PossessionGranter) DrainChangedStoryWeaponIds() []int32 {
	ids := g.LastChangedStoryWeaponIds
	g.LastChangedStoryWeaponIds = nil
	return ids
}

func (g *PossessionGranter) GrantFull(user *UserState, possessionType model.PossessionType, possessionId, count int32, nowMillis int64) {
	switch possessionType {
	case model.PossessionTypeCostume, model.PossessionTypeCostumeEnhanced:
		g.GrantCostume(user, possessionId, nowMillis)
	case model.PossessionTypeWeapon, model.PossessionTypeWeaponEnhanced:
		g.GrantWeapon(user, possessionId, nowMillis)
	default:
		GrantPossession(user, possessionType, possessionId, count)
	}
}

func (g *PossessionGranter) GrantCostume(user *UserState, costumeId int32, nowMillis int64) {
	for _, row := range user.Costumes {
		if row.CostumeId == costumeId {
			return
		}
	}
	if cm, ok := g.CostumeById[costumeId]; ok {
		if _, exists := user.Characters[cm.CharacterId]; !exists {
			user.Characters[cm.CharacterId] = CharacterState{
				CharacterId: cm.CharacterId,
				Level:       1,
			}
		}
	}
	key := uuid.New().String()
	user.Costumes[key] = CostumeState{
		UserCostumeUuid:     key,
		CostumeId:           costumeId,
		Level:               1,
		HeadupDisplayViewId: 1,
		AcquisitionDatetime: nowMillis,
	}
	user.CostumeActiveSkills[key] = CostumeActiveSkillState{
		UserCostumeUuid:     key,
		Level:               1,
		AcquisitionDatetime: nowMillis,
	}
}

func (g *PossessionGranter) GrantWeapon(user *UserState, weaponId int32, nowMillis int64) {
	key := uuid.New().String()
	user.Weapons[key] = WeaponState{
		UserWeaponUuid:      key,
		WeaponId:            weaponId,
		Level:               1,
		AcquisitionDatetime: nowMillis,
	}
	if _, exists := user.WeaponNotes[weaponId]; !exists {
		user.WeaponNotes[weaponId] = WeaponNoteState{
			WeaponId:                 weaponId,
			MaxLevel:                 1,
			MaxLimitBreakCount:       0,
			FirstAcquisitionDatetime: nowMillis,
			LatestVersion:            nowMillis,
		}
	}
	weapon, ok := g.WeaponById[weaponId]
	if !ok {
		return
	}

	g.populateWeaponSkillsAbilities(user, key, weapon)
	if weapon.WeaponStoryReleaseConditionGroupId != 0 {
		changed := false
		for _, cond := range g.ReleaseConditions[weapon.WeaponStoryReleaseConditionGroupId] {
			switch cond.WeaponStoryReleaseConditionType {
			case model.WeaponStoryReleaseConditionTypeAcquisition:
				if grantWeaponStoryUnlock(user, weaponId, cond.StoryIndex, nowMillis) {
					changed = true
				}
			case model.WeaponStoryReleaseConditionTypeQuestClear:
				if qs, ok := user.Quests[cond.ConditionValue]; ok && qs.QuestStateType == model.UserQuestStateTypeCleared {
					if grantWeaponStoryUnlock(user, weaponId, cond.StoryIndex, nowMillis) {
						changed = true
					}
				}
			}
		}
		if changed {
			g.LastChangedStoryWeaponIds = append(g.LastChangedStoryWeaponIds, weaponId)
		}
	}
}

func (g *PossessionGranter) populateWeaponSkillsAbilities(user *UserState, weaponUuid string, weapon WeaponRef) {
	if slots, ok := g.WeaponSkillSlots[weapon.WeaponSkillGroupId]; ok {
		skills := make([]WeaponSkillState, len(slots))
		for i, slot := range slots {
			skills[i] = WeaponSkillState{
				UserWeaponUuid: weaponUuid,
				SlotNumber:     slot,
				Level:          1,
			}
		}
		user.WeaponSkills[weaponUuid] = skills
	}
	if slots, ok := g.WeaponAbilitySlots[weapon.WeaponAbilityGroupId]; ok {
		abilities := make([]WeaponAbilityState, len(slots))
		for i, slot := range slots {
			abilities[i] = WeaponAbilityState{
				UserWeaponUuid: weaponUuid,
				SlotNumber:     slot,
				Level:          1,
			}
		}
		user.WeaponAbilities[weaponUuid] = abilities
	}
}

func GrantWeaponStoryUnlock(user *UserState, weaponId, storyIndex int32, nowMillis int64) bool {
	return grantWeaponStoryUnlock(user, weaponId, storyIndex, nowMillis)
}

func grantWeaponStoryUnlock(user *UserState, weaponId, storyIndex int32, nowMillis int64) bool {
	hasWeapon := false
	for _, row := range user.Weapons {
		if row.WeaponId == weaponId {
			hasWeapon = true
			break
		}
	}
	if !hasWeapon {
		log.Printf("[grantWeaponStoryUnlock] skipping weaponId=%d (weapon not in user.Weapons)", weaponId)
		return false
	}
	if user.WeaponStories == nil {
		user.WeaponStories = make(map[int32]WeaponStoryState)
	}
	cur := user.WeaponStories[weaponId]
	if storyIndex <= cur.ReleasedMaxStoryIndex {
		return false
	}
	user.WeaponStories[weaponId] = WeaponStoryState{
		WeaponId:              weaponId,
		ReleasedMaxStoryIndex: storyIndex,
		LatestVersion:         nowMillis,
	}
	return true
}

func EnsureDefaultDeck(user *UserState, nowMillis int64) {
	if len(user.Costumes) == 0 || len(user.Decks) > 0 {
		return
	}

	const rionCostumeId = int32(10100)
	const rionWeaponId = int32(101001)

	var costumeUuid, weaponUuid string
	for k, v := range user.Costumes {
		if v.CostumeId == rionCostumeId {
			costumeUuid = k
			break
		}
	}
	for k, v := range user.Weapons {
		if v.WeaponId == rionWeaponId {
			weaponUuid = k
			break
		}
	}

	dcUuid := uuid.New().String()
	user.DeckCharacters[dcUuid] = DeckCharacterState{
		UserDeckCharacterUuid: dcUuid,
		UserCompanionUuid:     "",
		UserCostumeUuid:       costumeUuid,
		MainUserWeaponUuid:    weaponUuid,
		Power:                 100,
		LatestVersion:         nowMillis,
	}
	user.Decks[DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 1}] = DeckState{
		DeckType:                model.DeckTypeQuest,
		UserDeckNumber:          1,
		UserDeckCharacterUuid01: dcUuid,
		Name:                    "Deck 1",
		Power:                   100,
		LatestVersion:           nowMillis,
	}

	if _, exists := user.DeckTypeNotes[model.DeckTypeQuest]; !exists {
		user.DeckTypeNotes[model.DeckTypeQuest] = DeckTypeNoteState{
			DeckType:      model.DeckTypeQuest,
			MaxDeckPower:  100,
			LatestVersion: nowMillis,
		}
	}
}

func FirstSortedKey[V any](m map[string]V) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys[0]
}

func ApplyDeckReplacement(user *UserState, deckType model.DeckType, userDeckNumber int32, slots []DeckCharacterInput, nowMillis int64) {
	deckKey := DeckKey{DeckType: deckType, UserDeckNumber: userDeckNumber}
	deck := user.Decks[deckKey]
	deck.DeckType = deckType
	deck.UserDeckNumber = userDeckNumber
	if deck.Name == "" {
		deck.Name = fmt.Sprintf("Deck %d", userDeckNumber)
	}
	if deck.Power == 0 {
		deck.Power = 100
	}

	uuidPtrs := []*string{&deck.UserDeckCharacterUuid01, &deck.UserDeckCharacterUuid02, &deck.UserDeckCharacterUuid03}
	for i, uuidPtr := range uuidPtrs {
		if i >= len(slots) || slots[i].UserCostumeUuid == "" {
			*uuidPtr = ""
			continue
		}
		slot := slots[i]
		dcUuid := *uuidPtr
		if dcUuid == "" {
			dcUuid = uuid.New().String()
		}
		dc := user.DeckCharacters[dcUuid]
		dc.UserDeckCharacterUuid = dcUuid
		dc.UserCostumeUuid = slot.UserCostumeUuid
		dc.MainUserWeaponUuid = slot.MainUserWeaponUuid
		dc.UserCompanionUuid = slot.UserCompanionUuid
		dc.UserThoughtUuid = slot.UserThoughtUuid
		dc.DressupCostumeId = slot.DressupCostumeId
		dc.LatestVersion = nowMillis
		user.DeckCharacters[dcUuid] = dc
		user.DeckSubWeapons[dcUuid] = slot.SubWeaponUuids
		user.DeckParts[dcUuid] = slot.PartsUuids
		*uuidPtr = dcUuid
	}

	deck.LatestVersion = nowMillis
	user.Decks[deckKey] = deck
}
