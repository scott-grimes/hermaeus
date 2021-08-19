package main

import (
    "context"
    "time"
    "example.com/m/v2/store"
    "example.com/m/v2/entry"
    "bytes"
)

// ORBIT_DB_IDENTITY
// ORBIT_DB_ADDRESS

func main() {
  s, err := store.NewStore(&store.CreateStoreOptions{
    Role: store.WorkerRole,
  })
  if err != nil {
    panic(err)
  }
  // would normally be a pdf or w/e
  ipfsRef, err := s.SaveDocument(context.TODO(), bytes.NewReader([]byte("foobar")))
  if err != nil {
    panic(err)
  }

  err = s.Put(entry.Entry{
    DocId: "someid",
    Content: "{\"somekey\":\"someval\"}",
    IpfsRef: ipfsRef,
    }, context.TODO())
  if err != nil {
    panic(err)
  }
  time.Sleep(time.Duration(1<<63 - 1))
}
