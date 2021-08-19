package main

import (
    "context"
    "time"
    "example.com/m/v2/store"
    "fmt"
)

func main() {
  s, err := store.NewStore(&store.CreateStoreOptions{
    Role: store.LeecherRole,
  })
  if err != nil {
    panic(err)
  }
  time.Sleep(time.Second * 5)
  ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
  blerg := s.RequestDoc("someid", ctx)
  fmt.Println("got here")
  fmt.Println(blerg)
}
