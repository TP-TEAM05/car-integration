package redis

import (
	"context"
	"encoding/json"
	"fmt"

	statistics "car-integration/services/statistics"
)

func GetNetworkStats(key string) *statistics.NetworkStats {
	ctx := context.Background()
	db := GetDB() // Get the Redis client

	// Get the serialized data from Redis
	serialized, err := db.Get(ctx, key).Result()
	if err != nil {
		fmt.Println("Error getting NetworkStats from Redis:", err)
		return nil
	}

	// Deserialize the data
	stats := &statistics.NetworkStats{}
	err = json.Unmarshal([]byte(serialized), stats)
	if err != nil {
		fmt.Println("Error deserializing NetworkStats:", err)
		return nil
	}

	return stats
}

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
