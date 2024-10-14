package redis

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"
	r "github.com/redis/go-redis/v9"
)

var DB *r.Client

func Init() {
	DB = r.NewClient(&r.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func GetDB() *r.Client {
	return DB
}

func HealthCheck() bool {
	ctx := context.Background()
	pong, err := DB.Ping(ctx).Result()
	if err != nil {
		sentry.CaptureException(err)
		fmt.Println("Redis health check failed")
		return false
	}
	fmt.Println("Redis health check passed: ", pong)
	return true
}
