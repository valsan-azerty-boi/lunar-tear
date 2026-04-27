package masterdata

import (
	"fmt"
	"log"
	"slices"
	"sort"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

type GachaPoolItem struct {
	PossessionType int32
	PossessionId   int32
	RarityType     model.RarityType
	CharacterId    int32
}

type FeaturedSet struct {
	Costumes []GachaPoolItem
	Weapons  []GachaPoolItem
}

type BannerPool struct {
	CostumesByRarity map[int32][]GachaPoolItem
	WeaponsByRarity  map[int32][]GachaPoolItem
	Featured         []GachaPoolItem
}

type ShopFeaturedEntry struct {
	CostumeId int32
	WeaponId  int32
}

type GachaCatalog struct {
	CostumesByRarity    map[int32][]GachaPoolItem
	WeaponsByRarity     map[int32][]GachaPoolItem
	Materials           []GachaPoolItem
	CostumeById         map[int32]GachaPoolItem
	WeaponById          map[int32]GachaPoolItem
	CostumeWeaponMap    map[int32]int32 // costumeId -> paired weaponId
	FeaturedByGacha     map[int32]FeaturedSet
	BannerPools         map[int32]*BannerPool
	ShopFeaturedByMedal map[int32][]ShopFeaturedEntry // consumableId -> paired entries
}

func LoadGachaPool() (*GachaCatalog, error) {
	costumes, err := utils.ReadTable[EntityMCostume]("m_costume")
	if err != nil {
		return nil, fmt.Errorf("load costume table: %w", err)
	}
	weapons, err := utils.ReadTable[EntityMWeapon]("m_weapon")
	if err != nil {
		return nil, fmt.Errorf("load weapon table: %w", err)
	}
	catalogCostumes, err := utils.ReadTable[EntityMCatalogCostume]("m_catalog_costume")
	if err != nil {
		return nil, fmt.Errorf("load catalog costume table: %w", err)
	}
	catalogWeapons, err := utils.ReadTable[EntityMCatalogWeapon]("m_catalog_weapon")
	if err != nil {
		return nil, fmt.Errorf("load catalog weapon table: %w", err)
	}
	materials, err := utils.ReadTable[EntityMMaterial]("m_material")
	if err != nil {
		return nil, fmt.Errorf("load material table: %w", err)
	}
	evoGroupRows, err := utils.ReadTable[EntityMWeaponEvolutionGroup]("m_weapon_evolution_group")
	if err != nil {
		return nil, fmt.Errorf("load weapon evolution group table: %w", err)
	}
	evolvedWeapons := buildEvolvedWeaponSet(evoGroupRows)

	catalogCostumeSet := make(map[int32]bool, len(catalogCostumes))
	costumeTermId := make(map[int32]int32, len(catalogCostumes))
	for _, c := range catalogCostumes {
		catalogCostumeSet[c.CostumeId] = true
		costumeTermId[c.CostumeId] = c.CatalogTermId
	}
	catalogWeaponSet := make(map[int32]bool, len(catalogWeapons))
	for _, w := range catalogWeapons {
		catalogWeaponSet[w.WeaponId] = true
	}

	costumeWeaponType := make(map[int32]int32, len(costumes))
	for _, c := range costumes {
		costumeWeaponType[c.CostumeId] = c.SkillfulWeaponType
	}

	weaponTypeById := make(map[int32]int32, len(weapons))
	weaponRarityById := make(map[int32]int32, len(weapons))
	restrictedWeapons := make(map[int32]bool)
	for _, w := range weapons {
		weaponTypeById[w.WeaponId] = w.WeaponType
		weaponRarityById[w.WeaponId] = w.RarityType
		if w.IsRestrictDiscard {
			restrictedWeapons[w.WeaponId] = true
		}
	}

	pool := &GachaCatalog{
		CostumesByRarity: make(map[int32][]GachaPoolItem),
		WeaponsByRarity:  make(map[int32][]GachaPoolItem),
		CostumeById:      make(map[int32]GachaPoolItem),
		WeaponById:       make(map[int32]GachaPoolItem),
		CostumeWeaponMap: make(map[int32]int32),
		FeaturedByGacha:  make(map[int32]FeaturedSet),
	}

	for _, c := range costumes {
		if !catalogCostumeSet[c.CostumeId] {
			continue
		}
		if c.RarityType < model.RaritySRare {
			continue
		}
		item := GachaPoolItem{
			PossessionType: int32(model.PossessionTypeCostume),
			PossessionId:   c.CostumeId,
			RarityType:     c.RarityType,
			CharacterId:    c.CharacterId,
		}
		pool.CostumesByRarity[c.RarityType] = append(pool.CostumesByRarity[c.RarityType], item)
		pool.CostumeById[c.CostumeId] = item
	}

	restrictedCount := 0
	for _, w := range weapons {
		if !catalogWeaponSet[w.WeaponId] {
			continue
		}
		if evolvedWeapons[w.WeaponId] {
			continue
		}
		item := GachaPoolItem{
			PossessionType: int32(model.PossessionTypeWeapon),
			PossessionId:   w.WeaponId,
			RarityType:     w.RarityType,
		}
		pool.WeaponById[w.WeaponId] = item
		if w.IsRestrictDiscard {
			restrictedCount++
			continue
		}
		pool.WeaponsByRarity[w.RarityType] = append(pool.WeaponsByRarity[w.RarityType], item)
	}

	log.Printf("[GachaPool] excluded %d evolved weapons, %d restricted weapons from pool", len(evolvedWeapons), restrictedCount)

	type weaponKey struct {
		TermId     int32
		WeaponType int32
		Rarity     int32
	}
	weaponsByKey := make(map[weaponKey][]int32)
	for _, cw := range catalogWeapons {
		if evolvedWeapons[cw.WeaponId] || restrictedWeapons[cw.WeaponId] {
			continue
		}
		wt := weaponTypeById[cw.WeaponId]
		r := weaponRarityById[cw.WeaponId]
		if wt == 0 || r < model.RaritySRare {
			continue
		}
		k := weaponKey{TermId: cw.CatalogTermId, WeaponType: wt, Rarity: r}
		weaponsByKey[k] = append(weaponsByKey[k], cw.WeaponId)
	}
	for k, ids := range weaponsByKey {
		slices.Sort(ids)
		weaponsByKey[k] = ids
	}

	exact, pattern, bestGuess := 0, 0, 0
	for costumeId, item := range pool.CostumeById {
		tid := costumeTermId[costumeId]
		wt := costumeWeaponType[costumeId]
		k := weaponKey{TermId: tid, WeaponType: wt, Rarity: item.RarityType}
		candidates := weaponsByKey[k]
		if len(candidates) == 0 {
			continue
		}
		if len(candidates) == 1 {
			pool.CostumeWeaponMap[costumeId] = candidates[0]
			exact++
			continue
		}
		idPattern := costumeId*10 + 1
		found := false
		for _, wid := range candidates {
			if wid == idPattern {
				pool.CostumeWeaponMap[costumeId] = wid
				pattern++
				found = true
				break
			}
		}
		if !found {
			pool.CostumeWeaponMap[costumeId] = candidates[0]
			bestGuess++
		}
	}
	log.Printf("[GachaPool] costume-weapon pairing: %d exact, %d id-pattern, %d best-guess, %d total",
		exact, pattern, bestGuess, len(pool.CostumeWeaponMap))

	for _, m := range materials {
		pool.Materials = append(pool.Materials, GachaPoolItem{
			PossessionType: int32(model.PossessionTypeMaterial),
			PossessionId:   m.MaterialId,
			RarityType:     m.RarityType,
		})
	}

	return pool, nil
}

func (pool *GachaCatalog) BuildShopFeatured(shop *ShopCatalog) {
	pool.ShopFeaturedByMedal = make(map[int32][]ShopFeaturedEntry)
	shopPairs := 0
	for _, cells := range shop.ExchangeShopCells {
		consumableId := shop.Items[cells[0].ShopItemId].PriceId

		var entries []ShopFeaturedEntry
		for _, cell := range cells {
			contents := shop.Contents[cell.ShopItemId]
			var costumeId, weaponId int32
			for _, c := range contents {
				switch c.PossessionType {
				case int32(model.PossessionTypeCostume):
					costumeId = c.PossessionId
				case int32(model.PossessionTypeWeapon):
					weaponId = c.PossessionId
				}
			}
			if costumeId == 0 && weaponId == 0 {
				continue
			}
			entries = append(entries, ShopFeaturedEntry{CostumeId: costumeId, WeaponId: weaponId})
			if costumeId != 0 && weaponId != 0 {
				pool.CostumeWeaponMap[costumeId] = weaponId
				shopPairs++
			}
		}
		if len(entries) > 0 {
			pool.ShopFeaturedByMedal[consumableId] = entries
		}
	}
	log.Printf("[GachaPool] shop featured: %d consumables, %d costume-weapon pairs overridden", len(pool.ShopFeaturedByMedal), shopPairs)
}

func (pool *GachaCatalog) PruneUnpairedCostumes() {
	pruned := 0
	for costumeId := range pool.CostumeById {
		if _, ok := pool.CostumeWeaponMap[costumeId]; !ok {
			delete(pool.CostumeById, costumeId)
			pruned++
		}
	}
	for rarity, items := range pool.CostumesByRarity {
		filtered := items[:0]
		for _, item := range items {
			if _, ok := pool.CostumeWeaponMap[item.PossessionId]; ok {
				filtered = append(filtered, item)
			}
		}
		pool.CostumesByRarity[rarity] = filtered
	}
	log.Printf("[GachaPool] pruned %d unpaired costumes", pruned)
}

func (pool *GachaCatalog) BuildFeaturedMapping(entries []store.GachaCatalogEntry) {
	matched := 0
	for _, entry := range entries {
		if entry.MedalConsumableItemId == 0 {
			continue
		}
		shopEntries, ok := pool.ShopFeaturedByMedal[entry.MedalConsumableItemId]
		if !ok || len(shopEntries) == 0 {
			continue
		}

		seenCostume := make(map[int32]bool)
		linkedWeapons := make(map[int32]bool)
		var costumes []GachaPoolItem
		for _, se := range shopEntries {
			if se.CostumeId != 0 && !seenCostume[se.CostumeId] {
				costumes = append(costumes, pool.CostumeById[se.CostumeId])
				seenCostume[se.CostumeId] = true
				linkedWeapons[se.WeaponId] = true
			}
		}

		seenWeapon := make(map[int32]bool)
		var weapons []GachaPoolItem
		for _, se := range shopEntries {
			if se.WeaponId != 0 && !linkedWeapons[se.WeaponId] && !seenWeapon[se.WeaponId] {
				if item, ok := pool.WeaponById[se.WeaponId]; ok {
					weapons = append(weapons, item)
					seenWeapon[se.WeaponId] = true
				}
			}
		}

		pool.FeaturedByGacha[entry.GachaId] = FeaturedSet{Costumes: costumes, Weapons: weapons}
		matched++
	}
	log.Printf("[GachaPool] featured mapping: %d/%d banners matched via shop", matched, len(entries))
}

func (pool *GachaCatalog) BuildBannerPools(entries []store.GachaCatalogEntry) {
	allFeaturedCostumes := make(map[int32]bool)
	allFeaturedWeapons := make(map[int32]bool)
	for _, fs := range pool.FeaturedByGacha {
		for _, c := range fs.Costumes {
			allFeaturedCostumes[c.PossessionId] = true
			allFeaturedWeapons[pool.CostumeWeaponMap[c.PossessionId]] = true
		}
		for _, w := range fs.Weapons {
			allFeaturedWeapons[w.PossessionId] = true
		}
	}

	commonCostumes := make(map[int32][]GachaPoolItem)
	for rarity, items := range pool.CostumesByRarity {
		for _, item := range items {
			if !allFeaturedCostumes[item.PossessionId] {
				commonCostumes[rarity] = append(commonCostumes[rarity], item)
			}
		}
	}
	commonWeapons := make(map[int32][]GachaPoolItem)
	for rarity, items := range pool.WeaponsByRarity {
		for _, item := range items {
			if !allFeaturedWeapons[item.PossessionId] {
				commonWeapons[rarity] = append(commonWeapons[rarity], item)
			}
		}
	}

	commonPool := &BannerPool{
		CostumesByRarity: commonCostumes,
		WeaponsByRarity:  commonWeapons,
	}

	pool.BannerPools = make(map[int32]*BannerPool)
	for _, entry := range entries {
		fs, hasFeatured := pool.FeaturedByGacha[entry.GachaId]
		if !hasFeatured {
			pool.BannerPools[entry.GachaId] = commonPool
			continue
		}

		var allFeatured []GachaPoolItem
		bannerCostumes := make(map[int32][]GachaPoolItem)
		for rarity, items := range commonCostumes {
			bannerCostumes[rarity] = append(bannerCostumes[rarity], items...)
		}
		bannerWeapons := make(map[int32][]GachaPoolItem)
		for rarity, items := range commonWeapons {
			bannerWeapons[rarity] = append(bannerWeapons[rarity], items...)
		}
		for _, c := range fs.Costumes {
			bannerCostumes[c.RarityType] = append(bannerCostumes[c.RarityType], c)
			allFeatured = append(allFeatured, c)
			wid := pool.CostumeWeaponMap[c.PossessionId]
			w := pool.WeaponById[wid]
			bannerWeapons[w.RarityType] = append(bannerWeapons[w.RarityType], w)
			allFeatured = append(allFeatured, w)
		}
		for _, w := range fs.Weapons {
			bannerWeapons[w.RarityType] = append(bannerWeapons[w.RarityType], w)
			allFeatured = append(allFeatured, w)
		}

		pool.BannerPools[entry.GachaId] = &BannerPool{
			CostumesByRarity: bannerCostumes,
			WeaponsByRarity:  bannerWeapons,
			Featured:         allFeatured,
		}
	}

	log.Printf("[GachaPool] banner pools: %d banners, %d featured costumes stripped, %d featured weapons stripped",
		len(pool.BannerPools), len(allFeaturedCostumes), len(allFeaturedWeapons))
}

func buildEvolvedWeaponSet(rows []EntityMWeaponEvolutionGroup) map[int32]bool {
	grouped := make(map[int32][]EntityMWeaponEvolutionGroup)
	for _, r := range rows {
		grouped[r.WeaponEvolutionGroupId] = append(grouped[r.WeaponEvolutionGroupId], r)
	}
	evolved := make(map[int32]bool)
	for _, chain := range grouped {
		sort.Slice(chain, func(i, j int) bool {
			return chain[i].EvolutionOrder < chain[j].EvolutionOrder
		})
		for i := 1; i < len(chain); i++ {
			evolved[chain[i].WeaponId] = true
		}
	}
	return evolved
}
