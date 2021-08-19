package store

import (
  orbitdb "berty.tech/go-orbit-db"
  "context"
)


type SubStore struct {
  Store orbitdb.KeyValueStore
  // mapping of prefixes to signatures,
  // sent by a replicator in response to a RequestStore request
  // each entry is ResponseStore jsonencoded
  PubSub PubSub
  Topic string

}

func (s *Store) GetRootStore() SubStore {
  return s.rootStore
}

func (s *Store) GetRootIpfsRef() string {
  return string(s.GetMain().All()["root"])
}

func (s *Store) GetMain() orbitdb.KeyValueStore {
  return s.main
}

func (s *Store) getSubstore(dbAddress string) (SubStore, bool) {
  s.SubStoresLock.RLock()
  defer s.SubStoresLock.RUnlock()
  var substore SubStore
  var exists bool
  substore, exists = s.SubStores[dbAddress];
  return substore, exists
}

func (s *Store) setSubstore(dbAddress string, substore *SubStore) {
  s.SubStoresLock.Lock()
  defer s.SubStoresLock.Unlock()
  s.SubStores[dbAddress] = *substore
}

func (s *Store) getSubstoreAddress(prefix string) (string, bool) {
  s.SubStoreOfLock.RLock()
  defer s.SubStoreOfLock.RUnlock()
  var address string
  var exists bool
  address, exists = s.SubStoreOf[prefix];
  return address, exists
}

func (s *Store) NewSubStore(substore orbitdb.KeyValueStore, isRoot bool) (*SubStore, error){
  topicName := substore.Address().String() + s.ChannelSuffix
  result := SubStore{
    Store: substore,
    Topic: topicName,
  }
  if s.Role == ReplicatorRole || s.Role == LeecherRole || isRoot {
    ps := s.NewPubSub(substore)
    topic, err := ps.TopicSubscribe(context.TODO(), topicName)
    if err != nil {
      return nil, err
    }
    err = ps.ListenForRequestDoc(context.TODO(),topic, *s)
    if err != nil {
      return nil, err
    }
    result.PubSub = ps
  }
  return &result, nil
}

func (s *SubStore) Close() {
  // todo rm pubsub+topic, close substore
  s.Close()
}
