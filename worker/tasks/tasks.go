package tasks

import (
  "example.com/m/v2/util"
  "github.com/hibiken/asynq"
  "encoding/json"
)

const (
  TypeDocFetch   = "doc:fetch"
)

type DocFetchPayload struct {
    DocId string
}

func NewDocFetchTask(docId string) (*asynq.Task, error) {
    payload, err := json.Marshal(DocFetchPayload{DocId: docId})
    if err != nil {
        return nil, err
    }
    return asynq.NewTask(TypeDocFetch, payload), nil
}

func NewAsyncRedisConnection() (asynq.RedisClientOpt) {
  REDIS_HOST := util.GetEnv("REDIS_HOST", "127.0.0.1:6379")

  return asynq.RedisClientOpt{
      Addr: REDIS_HOST,
      Password: "",
      DB: 0,
    }
}
