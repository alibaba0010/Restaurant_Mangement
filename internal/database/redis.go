package database

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis() {
    ctx := context.Background()
    
    rdb := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })
    
    // Ping to verify connection
    err := rdb.Ping(ctx).Err()
    if err != nil {
        panic(err)
    }
    
    // Set a key
    // err = rdb.Set(ctx, "key", "value", 0).Err()
    
    // // Get a key
    // val, err := rdb.Get(ctx, "key").Result()
}