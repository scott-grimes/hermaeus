package store

import (

  "context"

  "errors"
  "example.com/m/v2/util"

  "fmt"
  "encoding/hex"
  "encoding/json"

  "time"
  "example.com/m/v2/entry"

  orbitdb "berty.tech/go-orbit-db"
)
func (s *Store) Get(docId string, ctx context.Context) (*[]entry.Entry, error) {
  id := util.Md5hash(docId)

  var val string
  var cacheFetchErr error
  fmt.Println("fetching:" + id)
  if s.Cache != nil {
    val, cacheFetchErr = s.Cache.Get(ctx, id).Result()
  }

  if s.Cache == nil || cacheFetchErr != nil {
    var getErr error
    val, getErr = s.getRef(id, ctx)
    // todo handle err
    if getErr != nil {
      return nil, getErr
    }
    fmt.Println("retrieved:" + id + ":"  + val)
    if s.Cache != nil {
    cacheErr := s.Cache.Set(ctx, id, val, 60 * time.Second).Err();
      if cacheErr != nil {
        return nil, cacheErr
      }
      fmt.Println("storedcache:" + id + ":"  + val)
    }
  }
  entries := make([]entry.Entry,0)
	if err := json.Unmarshal([]byte(val), &entries); err != nil {
		return nil, err
	}
  return &entries, nil
}

func (s *Store) Put(e entry.Entry, ctx context.Context) error {
  id := util.Md5hash(e.DocId)

  fmt.Println("Storing " + id + ":" + e.IpfsRef)
  pubKeyBytes, err := s.PrivateKey.GetPublic().Raw()
  if err != nil {
    return err
  }
  e.Author = hex.EncodeToString(pubKeyBytes)
  b64Sig, err := entry.SignEntry(e, s.PrivateKey)
  if err != nil {
    return errors.New("Put failed, could not sign Entry")
  }
  e.Signature = b64Sig


  entries, err := s.Get(e.DocId,ctx)
  if err != nil {
    return err
  }

  var newEntries []entry.Entry

  if entries == nil || len(*entries) == 0 {
    newEntries = []entry.Entry{e}
  } else {
    newEntries = append(*entries, e)
  }
  value, err := json.Marshal(newEntries)
  if err != nil {
    return err
  }
  putErr := s.putRef(id,string(value),ctx)
  if putErr != nil {
    return putErr
  }
  if s.Config.EnableCache {
    if err := s.Cache.Set(ctx, id, value, 60 * time.Second).Err(); err != nil {
      return err
    }
    fmt.Println("stored value in cache: " + id + ":"  + string(value))
  }
  return nil
}

func (s *Store) findInSubstore(hash string, ctx context.Context, sub *SubStore) (string, error) {
  // todo break if not ascii
  curr := sub.Store
  s.Logger.Debug("entering findInSubstore for " + curr.Address().String())

  // do we see the current key?
  s.Logger.Debug("looking for hash: '" + hash + "' in " + curr.Address().String())

  val := string(curr.All()[hash])
  s.Logger.Debug("found: " + val)

  if val != "" {
    s.Logger.Debug("found item in leaf")
    return val, nil
  } else {
    s.Logger.Debug("did not find entry in leaf, returning []")
    return "[]", nil
  }
}

func (s *Store) putRef(hash string, v string, ctx context.Context) error {

    subStore, err := s.getSubstoreFromPrefix(hash[:KeyLength])
    if err != nil {
      return err
    }

  return s.putInSubstore(hash, v, ctx, subStore.Store)
}

func (s *Store) putInSubstore(hash string, value string, ctx context.Context, currentStore orbitdb.KeyValueStore) error {

  // this is a leaf, save value in store
  s.Logger.Debug("placing key:val " + hash + ":" + value + " in " + currentStore.Address().String())

  operation, err := currentStore.Put(ctx, hash, []byte(value) )
  if err != nil {
    return err
  }

  // todo here add the hash to a list. only return when its acked
  // via a "replicated" event
  s.Logger.Debug("working in database " + currentStore.Address().String())
  s.Logger.Debug("wrote hash " + *operation.GetKey())
  s.Logger.Debug("log id " + operation.GetEntry().GetLogID())
  // todo a chan here? look at examples

  // todo wait for replication before returning
  return nil
}
