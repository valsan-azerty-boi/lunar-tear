package utils

import (
	"encoding/json"
	"fmt"
	"lunar-tear/server/internal/masterdata/memorydb"
	"os"
	"path/filepath"
)

// ReadTable deserializes a master data table from the in-memory binary store.
// The key is the snake_case table name as it appears in the binary header
// (e.g. "m_weapon", "m_costume").
func ReadTable[T any](key string) ([]T, error) {
	return memorydb.ReadTable[T](key)
}

func EncodeJSONMaps(records ...map[string]any) (string, error) {
	jsonBytes, err := json.Marshal(records)
	if err != nil {
		return "", fmt.Errorf("json marshal maps: %w", err)
	}
	return string(jsonBytes), nil
}

// ReadJSON reads and deserializes a JSON master data file from assets/master_data/.
func ReadJSON[T any](filename string) ([]T, error) {
	path := filepath.Join("assets", "master_data", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var rows []T
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return rows, nil
}
