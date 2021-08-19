package store

import (
  // "encoding/json"

  "example.com/m/v2/broadcast"
  "example.com/m/v2/entry"
  "context"
  "reflect"
  "fmt"

  // "go.uber.org/zap"
)

func (s *Store) RequestDoc(docId string, ctx context.Context) *[]entry.Entry {

  rs := s.GetRootStore()
  ps := rs.PubSub

  // todo get root store topic
	// rootStoreTopic := s.GetRootStore

  topic, err := ps.TopicSubscribe(context.TODO(), rs.Topic)
	fmt.Println(err)
  err = ps.ListenForResponseDoc(docId, context.TODO(), topic, *s)
	fmt.Println(err)
	broadcast.BroadcastRequestDoc(broadcast.RequestDoc{
    DocId: docId,
    }, topic, context.TODO())
  responses := ps.Subscribe(ctx)

  // check if entry exists, return
  for response := range responses {
    event := &response
    fmt.Println(reflect.TypeOf(event))
    // entries := event.Entries
    fmt.Println("got message from my shit")
    fmt.Println(response)
    fmt.Println(reflect.TypeOf(response))
    entries := response.(event.EventResponseDocs).Entries
    fmt.Println(reflect.TypeOf(entries))
    docs, ok := response.([]entry.Entry)
    if !ok {
      continue
    }
    fmt.Println(docs)
    if len(docs)>0 && docs[0].DocId == docId {
      ok, err := entry.VerifyEntry(docs[0])
      if err != nil {
        fmt.Println(err)
      }
      if ok {
        fmt.Println("returning the found document")
        return &docs
      }
    }
  }
  // if context,  lose delfn return nil
  // defer deleteFn()
	return nil
}
