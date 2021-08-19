package store

import (
  "example.com/m/v2/util"
  "github.com/go-redis/redis/v8"
  "context"
)

func (s *Store) CreateCache() (error) {
  ctx, _ := context.WithCancel(context.Background())
  REDIS_HOST := util.GetEnv("REDIS_HOST", "127.0.0.1:6379")
  cache := redis.NewClient(&redis.Options{
     Addr: REDIS_HOST,
     Password: "",
     DB: 0,
  })
  if err := cache.Ping(ctx).Err(); err != nil {
     return err
  }
  s.Cache = cache
  return nil
}
