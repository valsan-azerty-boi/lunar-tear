package masterdata

import (
	"fmt"

	"lunar-tear/server/internal/utils"
)

type ConsumableItemCatalog struct {
	All map[int32]EntityMConsumableItem
}

func LoadConsumableItemCatalog() (*ConsumableItemCatalog, error) {
	rows, err := utils.ReadTable[EntityMConsumableItem]("m_consumable_item")
	if err != nil {
		return nil, fmt.Errorf("load consumable item table: %w", err)
	}

	catalog := &ConsumableItemCatalog{
		All: make(map[int32]EntityMConsumableItem, len(rows)),
	}
	for _, row := range rows {
		catalog.All[row.ConsumableItemId] = row
	}
	return catalog, nil
}
