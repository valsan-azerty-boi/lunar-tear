package masterdata

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/utils"
)

type CostumeMasterRow struct {
	CostumeId                        int32 `json:"CostumeId"`
	CharacterId                      int32 `json:"CharacterId"`
	SkillfulWeaponType               int32 `json:"SkillfulWeaponType"`
	RarityType                       int32 `json:"RarityType"`
	CostumeLimitBreakMaterialGroupId int32 `json:"CostumeLimitBreakMaterialGroupId"`
	CostumeActiveSkillGroupId        int32 `json:"CostumeActiveSkillGroupId"`
}

type costumeRarityRow struct {
	RarityType                                    int32 `json:"RarityType"`
	CostumeLimitBreakMaterialRarityGroupId        int32 `json:"CostumeLimitBreakMaterialRarityGroupId"`
	RequiredExpForLevelUpNumericalParameterMapId  int32 `json:"RequiredExpForLevelUpNumericalParameterMapId"`
	EnhancementCostByMaterialNumericalFunctionId  int32 `json:"EnhancementCostByMaterialNumericalFunctionId"`
	LimitBreakCostNumericalFunctionId             int32 `json:"LimitBreakCostNumericalFunctionId"`
	MaxLevelNumericalFunctionId                   int32 `json:"MaxLevelNumericalFunctionId"`
	ActiveSkillMaxLevelNumericalFunctionId        int32 `json:"ActiveSkillMaxLevelNumericalFunctionId"`
	ActiveSkillEnhancementCostNumericalFunctionId int32 `json:"ActiveSkillEnhancementCostNumericalFunctionId"`
}

type CostumeAwakenRow struct {
	CostumeId                        int32 `json:"CostumeId"`
	CostumeAwakenEffectGroupId       int32 `json:"CostumeAwakenEffectGroupId"`
	CostumeAwakenStepMaterialGroupId int32 `json:"CostumeAwakenStepMaterialGroupId"`
	CostumeAwakenPriceGroupId        int32 `json:"CostumeAwakenPriceGroupId"`
}

type costumeAwakenPriceRow struct {
	CostumeAwakenPriceGroupId int32 `json:"CostumeAwakenPriceGroupId"`
	AwakenStepLowerLimit      int32 `json:"AwakenStepLowerLimit"`
	Gold                      int32 `json:"Gold"`
}

type CostumeAwakenEffectRow struct {
	CostumeAwakenEffectGroupId int32 `json:"CostumeAwakenEffectGroupId"`
	AwakenStep                 int32 `json:"AwakenStep"`
	CostumeAwakenEffectType    int32 `json:"CostumeAwakenEffectType"`
	CostumeAwakenEffectId      int32 `json:"CostumeAwakenEffectId"`
}

type CostumeAwakenStatusUpRow struct {
	CostumeAwakenStatusUpGroupId int32 `json:"CostumeAwakenStatusUpGroupId"`
	SortOrder                    int32 `json:"SortOrder"`
	StatusKindType               int32 `json:"StatusKindType"`
	StatusCalculationType        int32 `json:"StatusCalculationType"`
	EffectValue                  int32 `json:"EffectValue"`
}

type CostumeAwakenItemAcquireRow struct {
	CostumeAwakenItemAcquireId int32 `json:"CostumeAwakenItemAcquireId"`
	PossessionType             int32 `json:"PossessionType"`
	PossessionId               int32 `json:"PossessionId"`
	Count                      int32 `json:"Count"`
}

type CostumeActiveSkillGroupRow struct {
	CostumeActiveSkillGroupId               int32 `json:"CostumeActiveSkillGroupId"`
	CostumeLimitBreakCountLowerLimit        int32 `json:"CostumeLimitBreakCountLowerLimit"`
	CostumeActiveSkillId                    int32 `json:"CostumeActiveSkillId"`
	CostumeActiveSkillEnhancementMaterialId int32 `json:"CostumeActiveSkillEnhancementMaterialId"`
}

type CostumeActiveSkillEnhanceMaterialRow struct {
	CostumeActiveSkillEnhancementMaterialId int32 `json:"CostumeActiveSkillEnhancementMaterialId"`
	SkillLevel                              int32 `json:"SkillLevel"`
	MaterialId                              int32 `json:"MaterialId"`
	Count                                   int32 `json:"Count"`
	SortOrder                               int32 `json:"SortOrder"`
}

type CostumeLotteryEffectRow struct {
	CostumeId                                 int32 `json:"CostumeId"`
	SlotNumber                                int32 `json:"SlotNumber"`
	CostumeLotteryEffectOddsGroupId           int32 `json:"CostumeLotteryEffectOddsGroupId"`
	CostumeLotteryEffectUnlockMaterialGroupId int32 `json:"CostumeLotteryEffectUnlockMaterialGroupId"`
	CostumeLotteryEffectDrawMaterialGroupId   int32 `json:"CostumeLotteryEffectDrawMaterialGroupId"`
	CostumeLotteryEffectReleaseScheduleId     int32 `json:"CostumeLotteryEffectReleaseScheduleId"`
}

type CostumeLotteryEffectMaterialGroupRow struct {
	CostumeLotteryEffectMaterialGroupId int32 `json:"CostumeLotteryEffectMaterialGroupId"`
	MaterialId                          int32 `json:"MaterialId"`
	Count                               int32 `json:"Count"`
	SortOrder                           int32 `json:"SortOrder"`
}

type CostumeLotteryEffectOddsRow struct {
	CostumeLotteryEffectOddsGroupId int32 `json:"CostumeLotteryEffectOddsGroupId"`
	OddsNumber                      int32 `json:"OddsNumber"`
	Weight                          int32 `json:"Weight"`
	CostumeLotteryEffectType        int32 `json:"CostumeLotteryEffectType"`
	CostumeLotteryEffectTargetId    int32 `json:"CostumeLotteryEffectTargetId"`
	RarityType                      int32 `json:"RarityType"`
}

type CostumeCatalog struct {
	Costumes               map[int32]CostumeMasterRow
	Materials              map[int32]MaterialRow
	ExpByRarity            map[int32][]int32
	EnhanceCostByRarity    map[int32]NumericalFunc
	MaxLevelByRarity       map[int32]NumericalFunc
	LimitBreakCostByRarity map[int32]NumericalFunc

	AwakenByCostumeId           map[int32]CostumeAwakenRow
	AwakenPriceByGroup          map[int32]int32
	AwakenEffectsByGroupAndStep map[int32]map[int32]CostumeAwakenEffectRow
	AwakenStatusUpByGroup       map[int32][]CostumeAwakenStatusUpRow
	AwakenItemAcquireById       map[int32]CostumeAwakenItemAcquireRow

	ActiveSkillGroupsByGroupId  map[int32][]CostumeActiveSkillGroupRow              // sorted by CostumeLimitBreakCountLowerLimit desc
	ActiveSkillEnhanceMats      map[[2]int32][]CostumeActiveSkillEnhanceMaterialRow // key: [enhancementMaterialId, skillLevel]
	ActiveSkillMaxLevelByRarity map[int32]NumericalFunc
	ActiveSkillCostByRarity     map[int32]NumericalFunc

	LotteryEffects    map[[2]int32]CostumeLotteryEffectRow             // key: [costumeId, slotNumber]
	LotteryEffectMats map[int32][]CostumeLotteryEffectMaterialGroupRow // key: materialGroupId (both unlock and draw)
	LotteryEffectOdds map[int32][]CostumeLotteryEffectOddsRow          // key: oddsGroupId
}

func LoadCostumeCatalog(matCatalog *MaterialCatalog) (*CostumeCatalog, error) {
	costumes, err := utils.ReadJSON[CostumeMasterRow]("EntityMCostumeTable.json")
	if err != nil {
		return nil, fmt.Errorf("load costume table: %w", err)
	}

	rarities, err := utils.ReadJSON[costumeRarityRow]("EntityMCostumeRarityTable.json")
	if err != nil {
		return nil, fmt.Errorf("load costume rarity table: %w", err)
	}

	paramMapRows, err := LoadParameterMap()
	if err != nil {
		return nil, err
	}

	funcResolver, err := LoadFunctionResolver()
	if err != nil {
		return nil, fmt.Errorf("load function resolver: %w", err)
	}

	awakenRows, err := utils.ReadJSON[CostumeAwakenRow]("EntityMCostumeAwakenTable.json")
	if err != nil {
		return nil, fmt.Errorf("load costume awaken table: %w", err)
	}
	awakenPriceRows, err := utils.ReadJSON[costumeAwakenPriceRow]("EntityMCostumeAwakenPriceGroupTable.json")
	if err != nil {
		return nil, fmt.Errorf("load costume awaken price table: %w", err)
	}
	awakenEffectRows, err := utils.ReadJSON[CostumeAwakenEffectRow]("EntityMCostumeAwakenEffectGroupTable.json")
	if err != nil {
		return nil, fmt.Errorf("load costume awaken effect table: %w", err)
	}
	awakenStatusUpRows, err := utils.ReadJSON[CostumeAwakenStatusUpRow]("EntityMCostumeAwakenStatusUpGroupTable.json")
	if err != nil {
		return nil, fmt.Errorf("load costume awaken status up table: %w", err)
	}
	awakenItemAcquireRows, err := utils.ReadJSON[CostumeAwakenItemAcquireRow]("EntityMCostumeAwakenItemAcquireTable.json")
	if err != nil {
		return nil, fmt.Errorf("load costume awaken item acquire table: %w", err)
	}

	activeSkillGroupRows, err := utils.ReadJSON[CostumeActiveSkillGroupRow]("EntityMCostumeActiveSkillGroupTable.json")
	if err != nil {
		return nil, fmt.Errorf("load costume active skill group table: %w", err)
	}
	activeSkillMatRows, err := utils.ReadJSON[CostumeActiveSkillEnhanceMaterialRow]("EntityMCostumeActiveSkillEnhancementMaterialTable.json")
	if err != nil {
		return nil, fmt.Errorf("load costume active skill enhancement material table: %w", err)
	}

	lotteryEffectRows, err := utils.ReadJSON[CostumeLotteryEffectRow]("EntityMCostumeLotteryEffectTable.json")
	if err != nil {
		return nil, fmt.Errorf("load costume lottery effect table: %w", err)
	}
	lotteryEffectMatRows, err := utils.ReadJSON[CostumeLotteryEffectMaterialGroupRow]("EntityMCostumeLotteryEffectMaterialGroupTable.json")
	if err != nil {
		return nil, fmt.Errorf("load costume lottery effect material group table: %w", err)
	}
	lotteryEffectOddsRows, err := utils.ReadJSON[CostumeLotteryEffectOddsRow]("EntityMCostumeLotteryEffectOddsGroupTable.json")
	if err != nil {
		return nil, fmt.Errorf("load costume lottery effect odds group table: %w", err)
	}

	catalog := &CostumeCatalog{
		Costumes:               make(map[int32]CostumeMasterRow, len(costumes)),
		Materials:              matCatalog.ByType[model.MaterialTypeCostumeEnhancement],
		ExpByRarity:            make(map[int32][]int32, len(rarities)),
		EnhanceCostByRarity:    make(map[int32]NumericalFunc, len(rarities)),
		MaxLevelByRarity:       make(map[int32]NumericalFunc, len(rarities)),
		LimitBreakCostByRarity: make(map[int32]NumericalFunc, len(rarities)),

		AwakenByCostumeId:           make(map[int32]CostumeAwakenRow, len(awakenRows)),
		AwakenPriceByGroup:          make(map[int32]int32, len(awakenPriceRows)),
		AwakenEffectsByGroupAndStep: make(map[int32]map[int32]CostumeAwakenEffectRow),
		AwakenStatusUpByGroup:       make(map[int32][]CostumeAwakenStatusUpRow),
		AwakenItemAcquireById:       make(map[int32]CostumeAwakenItemAcquireRow, len(awakenItemAcquireRows)),

		ActiveSkillGroupsByGroupId:  make(map[int32][]CostumeActiveSkillGroupRow),
		ActiveSkillEnhanceMats:      make(map[[2]int32][]CostumeActiveSkillEnhanceMaterialRow),
		ActiveSkillMaxLevelByRarity: make(map[int32]NumericalFunc, len(rarities)),
		ActiveSkillCostByRarity:     make(map[int32]NumericalFunc, len(rarities)),

		LotteryEffects:    make(map[[2]int32]CostumeLotteryEffectRow, len(lotteryEffectRows)),
		LotteryEffectMats: make(map[int32][]CostumeLotteryEffectMaterialGroupRow),
		LotteryEffectOdds: make(map[int32][]CostumeLotteryEffectOddsRow),
	}

	for _, row := range costumes {
		catalog.Costumes[row.CostumeId] = row
	}

	for _, r := range rarities {
		if _, ok := catalog.ExpByRarity[r.RarityType]; !ok {
			catalog.ExpByRarity[r.RarityType] = BuildExpThresholds(paramMapRows, r.RequiredExpForLevelUpNumericalParameterMapId)
		}
		if _, ok := catalog.EnhanceCostByRarity[r.RarityType]; !ok {
			if f, found := funcResolver.Resolve(r.EnhancementCostByMaterialNumericalFunctionId); found {
				catalog.EnhanceCostByRarity[r.RarityType] = f
			}
		}
		if _, ok := catalog.MaxLevelByRarity[r.RarityType]; !ok {
			if f, found := funcResolver.Resolve(r.MaxLevelNumericalFunctionId); found {
				catalog.MaxLevelByRarity[r.RarityType] = f
			}
		}
		if _, ok := catalog.LimitBreakCostByRarity[r.RarityType]; !ok {
			if f, found := funcResolver.Resolve(r.LimitBreakCostNumericalFunctionId); found {
				catalog.LimitBreakCostByRarity[r.RarityType] = f
			}
		}
		if _, ok := catalog.ActiveSkillMaxLevelByRarity[r.RarityType]; !ok {
			if f, found := funcResolver.Resolve(r.ActiveSkillMaxLevelNumericalFunctionId); found {
				catalog.ActiveSkillMaxLevelByRarity[r.RarityType] = f
			}
		}
		if _, ok := catalog.ActiveSkillCostByRarity[r.RarityType]; !ok {
			if f, found := funcResolver.Resolve(r.ActiveSkillEnhancementCostNumericalFunctionId); found {
				catalog.ActiveSkillCostByRarity[r.RarityType] = f
			}
		}
	}

	for _, row := range awakenRows {
		catalog.AwakenByCostumeId[row.CostumeId] = row
	}
	for _, row := range awakenPriceRows {
		catalog.AwakenPriceByGroup[row.CostumeAwakenPriceGroupId] = row.Gold
	}
	for _, row := range awakenEffectRows {
		m, ok := catalog.AwakenEffectsByGroupAndStep[row.CostumeAwakenEffectGroupId]
		if !ok {
			m = make(map[int32]CostumeAwakenEffectRow)
			catalog.AwakenEffectsByGroupAndStep[row.CostumeAwakenEffectGroupId] = m
		}
		m[row.AwakenStep] = row
	}
	for _, row := range awakenStatusUpRows {
		catalog.AwakenStatusUpByGroup[row.CostumeAwakenStatusUpGroupId] = append(
			catalog.AwakenStatusUpByGroup[row.CostumeAwakenStatusUpGroupId], row)
	}
	for _, row := range awakenItemAcquireRows {
		catalog.AwakenItemAcquireById[row.CostumeAwakenItemAcquireId] = row
	}

	for _, row := range activeSkillGroupRows {
		gid := row.CostumeActiveSkillGroupId
		catalog.ActiveSkillGroupsByGroupId[gid] = append(catalog.ActiveSkillGroupsByGroupId[gid], row)
	}
	for gid, rows := range catalog.ActiveSkillGroupsByGroupId {
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].CostumeLimitBreakCountLowerLimit > rows[j].CostumeLimitBreakCountLowerLimit
		})
		catalog.ActiveSkillGroupsByGroupId[gid] = rows
	}

	for _, row := range activeSkillMatRows {
		key := [2]int32{row.CostumeActiveSkillEnhancementMaterialId, row.SkillLevel}
		catalog.ActiveSkillEnhanceMats[key] = append(catalog.ActiveSkillEnhanceMats[key], row)
	}

	for _, row := range lotteryEffectRows {
		key := [2]int32{row.CostumeId, row.SlotNumber}
		catalog.LotteryEffects[key] = row
	}
	for _, row := range lotteryEffectMatRows {
		gid := row.CostumeLotteryEffectMaterialGroupId
		catalog.LotteryEffectMats[gid] = append(catalog.LotteryEffectMats[gid], row)
	}
	for _, row := range lotteryEffectOddsRows {
		gid := row.CostumeLotteryEffectOddsGroupId
		catalog.LotteryEffectOdds[gid] = append(catalog.LotteryEffectOdds[gid], row)
	}

	return catalog, nil
}
