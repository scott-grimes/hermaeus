package main

import (
  "example.com/m/v2/store"
  "time"
)

func main() {
  _, err := store.NewStore(&store.CreateStoreOptions{
    Role: store.ReplicatorRole,
  })
  if err != nil {
    panic(err)
  }
  time.Sleep(time.Duration(1<<63 - 1))
}
