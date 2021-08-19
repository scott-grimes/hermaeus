package snapshot

import (
  // "berty.tech/go-ipfs-log/identityprovider"
  // "encoding/base64"
  "github.com/ipfs/go-cid"
  // "encoding/hex"
  // dssync "github.com/ipfs/go-datastore/sync"
    "berty.tech/go-orbit-db/address"
  // "reflect"
  "errors"
  "fmt"
  // "github.com/libp2p/go-libp2p-core/crypto"
    // "path/filepath"
    leveldb "github.com/ipfs/go-ds-leveldb"
    "io/ioutil"
    "path"
  // "berty.tech/go-ipfs-log/keystore"
  "github.com/ipfs/go-datastore"
  // "unsafe"
  "strings"
  "berty.tech/go-orbit-db/iface"

  orbitdb "berty.tech/go-orbit-db"

  // "os"
  "context"
)

func (s *Store) injectCidIntoCache(dbAddress string, cidHash string) (error) {
  parsedAddress, err := address.Parse(dbAddress)
  // var flist []string
  if err != nil {
		fmt.Println(errors.New("unable to parse address " + parsedAddress.String()))
    panic(errors.New("unable to parse address " + parsedAddress.String()))
	}


  dbPath := path.Join(parsedAddress.GetRoot().String(), parsedAddress.GetPath())
	keyPath := path.Join(s.OrbitDirectory, dbPath)
  fmt.Println(keyPath)
  fmt.Println("should be /Users/name/git/hermaeus/.persist/replicator0/orbitKeystore/bafyreiblasu7qb42fnpexroahnkyziwzrk6kd6yg4neqo4apha3ix4ovie/hermaeus")

  // then
  ds2, lvdberr := leveldb.NewDatastore(keyPath, nil)
  if lvdberr != nil {
    panic(lvdberr)
  }


  cacheKey := datastore.NewKey(path.Join(parsedAddress.String(), "_manifest"))
  fmt.Println("cacheKey")
  fmt.Println(cacheKey)

  fmt.Println(ds2.Get(cacheKey))
  ds2.Put(cacheKey, []byte(parsedAddress.GetRoot().String()))
  fmt.Println(ds2.Get(cacheKey))

  file, fErr := s.GetDocument(context.TODO(), string(cidHash))
  fmt.Println(file)
  fmt.Println(fErr)
  _, fileBytesErr := ioutil.ReadAll(file)
  if fileBytesErr != nil {
    panic(fileBytesErr)
  }
  closeErr := ds2.Close()
  if closeErr != nil {
    return err
  }
  return nil
}

func (s *Store) PreloadCids(importedTar map[string]string) (error) {
  for dbAddress, cidHash := range importedTar {
    // strings starts with store?
    prefix := "snapshot/"
    if strings.HasPrefix(dbAddress, prefix){
      parsedAddress, err := address.Parse(dbAddress[:len(prefix)])
      // var flist []string
      if err != nil {
        fmt.Println(errors.New("unable to parse address " + parsedAddress.String()))
        panic(errors.New("unable to parse address " + parsedAddress.String()))
      }
      fmt.Println(parsedAddress.String())
      c , err := cid.Decode(cidHash)
      if err != nil {
        panic(err)
      }
      ctx, _ := context.WithCancel(context.Background())

      var replication = true
      var create = false
      var storeType = "keyvalue"

      store, err := s.Orbit.Open(ctx, dbAddress, &iface.CreateDBOptions{
          Replicate: &replication,
          Identity: s.Orbit.Identity(),
          Create: &create,
          StoreType: &storeType,
          LocalOnly: &create,
      })
      if err != nil {
        panic(err)
      }
      db, ok := store.(orbitdb.KeyValueStore)
      if !ok {
        return errors.New("unable to cast store to keyvalue")
      }
      var cidArry []cid.Cid
      cidArry = append(cidArry,c)
      db.LoadMoreFrom(context.TODO(), 100, cidArry)
      db.Close()
    }

  }
  return nil
}

// disable connections here?
func (s *Store) PreLoadOrbitDbStores(importedTar map[string]string) (error) {

  for dbAddress, cidHash := range importedTar {
    parsedAddress, err := address.Parse(dbAddress)
    // var flist []string
    if err != nil {
      fmt.Println(errors.New("unable to parse address " + parsedAddress.String()))
      panic(errors.New("unable to parse address " + parsedAddress.String()))
    }
    fmt.Println(parsedAddress.String())

    dbPath := path.Join(parsedAddress.GetRoot().String(), parsedAddress.GetPath())
  	keyPath := path.Join(s.OrbitDirectory, dbPath)
    ds2, lvdberr := leveldb.NewDatastore(keyPath, nil)
    if lvdberr != nil {
      panic(lvdberr)
    }
    cacheKey := datastore.NewKey(path.Join(parsedAddress.String(), "_manifest"))

    fmt.Println("cacheKey")
    fmt.Println(cacheKey)
    ds2.Put(cacheKey, []byte(parsedAddress.GetRoot().String()))
    ds2.Put(datastore.NewKey("snapshot"), []byte(cidHash))
    closeErr := ds2.Close()
    if closeErr != nil {
      return err
    }


    var replication = false
    var create = false
    var storeType = "keyvalue"
    tempStore, err := s.Orbit.Open(context.TODO(), dbAddress, &iface.CreateDBOptions{
      Replicate: &replication,
      Create: &create,
      StoreType: &storeType,
    })
    if err != nil {
      panic(err)
    }
    tempStore.Cache().Get(datastore.NewKey("snapshot"))
    tempStore.Cache().Put(datastore.NewKey("snapshot"), []byte(cidHash))
    tempStore.Cache().Get(datastore.NewKey("snapshot"))
    tempStore.Close()

    // ds, err := s.OrbitCache.Load(s.OrbitDirectory, parsedAddress)
    // if err != nil {
    //   fmt.Println(errors.New("unable to open Orbit Cache for " + parsedAddress.String()))
    //   panic(errors.New("unable to open Orbit Cache for " + parsedAddress.String()))
    // }

    // err = ds.Put(datastore.NewKey("snapshot"), []byte(cidHash))
    // if err != nil {
    //   fmt.Println(errors.New("unable to add snapshot data to cache for " + parsedAddress.String()))
    //   panic(errors.New("unable to add snapshot data to cache for " + parsedAddress.String()))
    // }
    // blerg, _ := ds.Get(datastore.NewKey("snapshot"))
    // fmt.Println("got from cache " + string(blerg))
    //
    //
    // fmt.Println(ds.Get(cacheKey))
    //
    // dsputerr := ds.Put(cacheKey, []byte(parsedAddress.GetRoot().String()))
    // fmt.Println(dsputerr)
    // dscachekey, _ := ds.Get(cacheKey)
    // fmt.Println("dscache key from datastore")
    // fmt.Println(string(dscachekey))
    // ds.Close()
  }
  return nil
}
