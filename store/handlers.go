package store
import (
	// p2pcore "github.com/libp2p/go-libp2p-core"
	"encoding/json"
	"go.uber.org/zap"
	"context"
	"example.com/m/v2/broadcast"
	"example.com/m/v2/entry"
	"fmt"
	// "github.com/patrickmn/go-cache"
)

func (s *Store) handleRequestDoc(payload string) []entry.Entry {
	fmt.Println("entered handleRequestDoc")
  var m broadcast.RequestDoc
	err := json.Unmarshal([]byte(payload), &m)
  if err != nil {
		s.Logger.Error("Unmarshal Message Error", zap.Error(err))
    return nil
  }
	entries, err := s.Get(m.DocId,context.TODO())
	if err != nil {
		s.Logger.Error("Get Error for docId: " + m.DocId, zap.Error(err))
    return nil
	}
	return *entries
}

func (s *Store) handleResponseDoc(payload string) []entry.Entry {
	fmt.Println("entered handleResponseDoc")
	entries := make([]entry.Entry,0)
	err := json.Unmarshal([]byte(payload), &entries)
  if err != nil {
		s.Logger.Error("Unmarshal Message Error", zap.Error(err))
    return nil
  }
	return entries
}
