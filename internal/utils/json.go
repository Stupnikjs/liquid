package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
)

func SavePoolEdgesJSON(edges []lqtypes.PoolEdge, path string) error {
	data, err := json.MarshalIndent(edges, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func LoadPoolEdgesJSON(path string) ([]lqtypes.PoolEdge, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var edges []lqtypes.PoolEdge
	if err := json.Unmarshal(data, &edges); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return edges, nil
}
