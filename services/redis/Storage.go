package redis

import (
	"context"
	"encoding/json"
	"fmt"

	statistics "car-integration/services/statistics"
)

func SaveNetworkStats(key string, stats *statistics.NetworkStats) error {
	ctx := context.Background()
	db := GetDB() // Get the Redis client

	// Serialize the stats to JSON for storing
	serialized, err := json.Marshal(stats)
	if err != nil {
		fmt.Println("Error serializing NetworkStats:", err)
		return err
	}

	// Save the serialized data to Redis
	err = db.Set(ctx, key, serialized, 0).Err()
	if err != nil {
		fmt.Println("Error saving NetworkStats to Redis:", err)
		return err
	}

	return nil
}
