package store

import (
  "berty.tech/go-ipfs-log/identityprovider"
  "berty.tech/go-orbit-db/address"
  "berty.tech/go-orbit-db/iface"
  "berty.tech/go-orbit-db/stores"
  "berty.tech/go-orbit-db/stores/basestore"
  // "bytes"
  "github.com/hibiken/asynq"
  "example.com/m/v2/worker/tasks"
  "context"
  // "github.com/ipfs/go-ipfs-files"
  // "io"
  "io/ioutil"
  // "math/rand"
  // "berty.tech/go-orbit-db/stores/operation"
  // "encoding/base64"
  "math/rand"
  "errors"
  "example.com/m/v2/util"
  // corepath "github.com/ipfs/interface-go-ipfs-core/path"
  // ipfslog "berty.tech/go-ipfs-log"
  "fmt"
  // "berty.tech/go-orbit-db/cache"
  "github.com/go-redis/redis/v8"
  "berty.tech/go-orbit-db/cache/cacheleveldown"
  "github.com/libp2p/go-libp2p-core/crypto"
  "encoding/json"
  // "github.com/ipfs/go-cid"
  // "github.com/ipfs/go-ipfs-config"
  // "github.com/ipfs/go-ipfs/core/coreapi"
  // "github.com/ipfs/go-ipfs/core/node/libp2p"
  // "github.com/ipfs/go-ipfs/plugin/loader"
  coreinterface "github.com/ipfs/interface-go-ipfs-core"
  repo "github.com/ipfs/go-ipfs/repo"
  // "github.com/ipfs/interface-go-ipfs-core/options"
  "go.uber.org/zap"
  "path/filepath"
  "sync"
  // "strings"
  "time"
  // "os"
  core "github.com/ipfs/go-ipfs/core"
  // fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"
  orbitdb "berty.tech/go-orbit-db"
  "example.com/m/v2/orbit"
  // "example.com/m/v2/substore"
)

// todo make this store, dedupe
const KeyLength = 4

// how many channels exist for requests?
const ChannelShards = 1

type Role int
const (
    ReplicatorRole Role = iota
    WorkerRole
    LeecherRole
)

type Store struct {
  Logger *zap.Logger

  Role Role

  Orbit orbitdb.OrbitDB
  OrbitDirectory string
  // OrbitCache cache.Interface

  // todo rm
  IsFirstRepoInit bool

  // Database Entrypoint
  main orbitdb.KeyValueStore

  // current root store
  rootStore SubStore

  BannerMessage string

  // mapping of in-use databaseAddresses to kvstores
  SubStores map[string]SubStore
  SubStoresLock sync.RWMutex

  // ipfsRef holding current list of substores
  SubStoreIpfsRef string

  // mapping of prefixes to databaseAddresses
  SubStoreOf map[string]string
  SubStoreOfLock sync.RWMutex

  // used by workers to sign items before storing in db
  PrivateKey crypto.PrivKey //Secp256k1PrivateKey

  // all channels are sharded, with ChannelShards num shards
  ChannelSuffix string

  Api coreinterface.CoreAPI
  Node *core.IpfsNode
  Repo repo.Repo
  // used by workers to coordinate work
  Cache *redis.Client
  AsyncClient *asynq.Client

  Config CreateStoreOptions
}

type CreateStoreOptions struct {
  Role Role
  Logger *zap.Logger

  // Used by Replicator
  EnableCache bool
  FractionPin int
  MaxGb int
  ManageWorkers bool

  // Used for debugging
  SkipOrbit bool
}

func (s *Store) GetUniqueSubstores() map[string]string {
  s.SubStoreOfLock.RLock()
  defer s.SubStoreOfLock.RUnlock()
  res := make(map[string]string)

  for prefix, address := range s.SubStoreOf {
    if prefixes, ok := res[address]; !ok {
      res[address] = prefix
    } else {
      res[address] = prefixes + "," + prefix
    }
  }
  return res
}

func (s *Store) updateSubstores() error {

  // todo, only update those stores which match parameters for following based on random id.

  if s.Role == LeecherRole {
    return nil
  }
  rs := s.GetRootStore().Store
  rootRaw := rs.All()
  s.Logger.Debug("entering updatesubstores with root store " + rs.Address().String())

  substoresIpfsRef, ok := rootRaw["substores"]
  if !ok {
    return errors.New("no substores ipfsref found in root database")
  }

  // todo if new ref is not equal to old ref unpin the old one, pin the new one
  err := s.Pin(context.TODO(),string(substoresIpfsRef))
  if err != nil {
    return err
  }
  jsonSubstores, err := s.GetDocument(context.TODO(), string(substoresIpfsRef))
  if err != nil {
    return err
  }
  newAddressesBytes, err := ioutil.ReadAll(jsonSubstores)
  if err != nil {
    return err
  }
  // prefix -> address map of substores
  var newSubstores map[string]string
  err = json.Unmarshal(newAddressesBytes, &newSubstores);
  if err != nil {
    return err
  }

  s.SubStoreOfLock.Lock()
  s.SubStoresLock.Lock()

  oldSubStoreOf := s.SubStoreOf
  var oldAddresses []string
  var newAddresses []string
  var addressesToDelete []string
  for oldAddress, _ := range s.SubStores {
    oldAddresses = append(oldAddresses, oldAddress)
  }
  fmt.Println("oldAddress")
  fmt.Println(len(oldAddresses))

  for prefix, newAddress := range newSubstores {
    newAddresses = append(newAddresses, newAddress)
    oldAddress, ok := oldSubStoreOf[prefix]
    if ok && oldAddress != newAddress {
      addressesToDelete = append(addressesToDelete, oldSubStoreOf[prefix])
      // "prefix " + prefix + "changed from " + oldSubStoreOf[prefix] + " to " + newAddress
    } else if !ok {
      // "prefix " + prefix + " initialized to " + newAddress
    }
    s.SubStoreOf[prefix] = newAddress
  }


  fmt.Println("addressesToDelete")
  fmt.Println(len(addressesToDelete))

  for _, oldDbAddress  := range addressesToDelete {
    sub, ok := s.SubStores[oldDbAddress]
    if ok {
      sub.Close()
      delete(s.SubStores,oldDbAddress);
    }
  }

  addressesToAdd := util.ANotInB(newAddresses, oldAddresses)
  fmt.Println("addressesToAdd")
  fmt.Println(len(addressesToAdd))

  // todo assert that prefixes in substoresOf == 16^4
  s.SubStoresLock.Unlock()
  s.SubStoreOfLock.Unlock()

  for _, newAddress := range addressesToAdd {
    _, err := s.openSubStore(newAddress)
    if err != nil {
      return err
    }
  }
  return nil
}

func (s *Store) loadNewRoot(dbAddress string) error {
  // todo rm context
  // open keyvalue store new root
  newRoot, err := s.openKeyValueStore(dbAddress)
  if err != nil {
    return err
  }
  ctx, _ := context.WithCancel(context.Background())
  fmt.Println("in load new root")

  substore, err := s.NewSubStore(newRoot, true)
  if err != nil {
    return err
  }
  fmt.Println("opened root")
  s.rootStore = *substore

  sub := newRoot.Subscribe(ctx)
  go func() {
    for e := range sub {
      switch e.(type) {
      case *stores.EventReplicated:
        s.Logger.Debug("root replicated")
        if s.Role != LeecherRole {
          _, err := basestore.SaveSnapshot(ctx, newRoot)
          if err != nil {
            s.Logger.Error(err.Error())
          }
        }
        newSubstoreIpfsRef, ok := newRoot.All()["substores"]
        if ok && string(newSubstoreIpfsRef) != s.SubStoreIpfsRef {
          err := s.updateSubstores()
          if err != nil {
            s.Logger.Error(err.Error())
          }
        }
        newBannerMessage, ok := newRoot.All()["bannermessage"]
        if ok && string(newBannerMessage) != s.BannerMessage {
          s.BannerMessage = string(newBannerMessage)
        }
        fmt.Println("passed eventreplicated")
      case *stores.EventReady, *stores.EventReplicateProgress:
        s.Logger.Debug("got root event ready/replicateprogress, updating substores")
        newSubstoreIpfsRef, ok := newRoot.All()["substores"]
        if ok && string(newSubstoreIpfsRef) != s.SubStoreIpfsRef {
          err := s.updateSubstores()
          if err != nil {
            s.Logger.Error(err.Error())
          }
        }
        newBannerMessage, ok := newRoot.All()["bannermessage"]
        if ok && string(newBannerMessage) != s.BannerMessage {
          s.BannerMessage = string(newBannerMessage)
        }
        fmt.Println("passed eventready/progress")
      }
    }
  }()
  err = newRoot.Load(ctx, -1)
  if err != nil {
    return err
  }
  err = s.updateSubstores()
  if err != nil {
    s.Logger.Info(err.Error())
  }
  return nil
}

func (s *Store) reconfigureRoot() error {
  // todo implement timeout
  // todo subscribe
  fmt.Println("entered reconfiguring root")
  root := string(s.GetMain().All()["root"])
  rs := s.GetRootStore()
  if root != "" {
    if rs.Store == nil {
      s.Logger.Debug("loading first root")
      err := s.loadNewRoot(root)
      if err != nil {
        return err
      } else {
        s.Logger.Debug("root added")
      }
    } else if rs.Store != nil && rs.Store.Address().String() != root {
      fmt.Println("new root detected")
      err := s.loadNewRoot(root)
      if err != nil {
        return err
      } else {
        s.Logger.Debug("root successfully changed")
        rs.Close()
      }
    } else if rs.Store.Address().String() == root {
    s.Logger.Debug("root address unchanged")
    }
  } else {
    s.Logger.Debug("root not found")
  }
  return nil
}

func (s *Store) getSubstoreFromPrefix(prefix string) (*SubStore, error) {

    substoreAddress, exists := s.getSubstoreAddress(prefix)
    if !exists {
      return nil, errors.New("no substore exists with prefix :" + prefix )
    }
    subStore, exists := s.getSubstore(substoreAddress)
    if !exists {
      return nil, errors.New("no substore exists with address: " + substoreAddress )
    }
    return &subStore, nil
}

func (s *Store) getRef(hash string, ctx context.Context) (string, error) {
  fmt.Println("entering getRef")

  subStore, err := s.getSubstoreFromPrefix(hash[:KeyLength])
  if err != nil {
    return "", err
  }

  return s.findInSubstore(hash, ctx, subStore)
}

func (s *Store) configureMain() error {
  // todo rm context
  ctx, _ := context.WithCancel(context.Background())
  if s.Role == ReplicatorRole {
    s.Logger.Debug("loading snapshot of main")
    err := s.GetMain().LoadFromSnapshot(ctx)
    if err != nil {
      s.Logger.Debug("could not load main base from snapshot")
    }
  }
  main := s.GetMain()
  sub := main.Subscribe(ctx)
  go func() {
    for e := range sub {
      switch e.(type) {
      case *stores.EventReplicated:
        s.Logger.Debug("main replicated: " + main.Address().String())
        if s.Role == ReplicatorRole {
          _, err := basestore.SaveSnapshot(ctx, main)
          if err != nil {
            fmt.Println(err)
          }
        }
        err := s.reconfigureRoot()
        if err != nil {
          fmt.Println(err)
        }
      }
    }
    }()
  err := s.GetMain().Load(ctx, -1)
  if err != nil {
    return err
  }
  err = s.reconfigureRoot()
  if err != nil {
    fmt.Println(err)
    return err
  }
  var rootStoreLoadAttempts = 0
  var rootStoreLoaded = false;
  for !rootStoreLoaded {
    if s.GetRootStore().Store != nil {
      return nil
    } else {
      rootStoreLoadAttempts = rootStoreLoadAttempts + 1
      s.Logger.Debug("root store not yet loaded...")
      time.Sleep(5 * time.Second)
    }
    if rootStoreLoadAttempts > 5 {
      return errors.New("exceeded root store load attempts")
    }
  }
  return nil
}

func (s *Store) openSubStore(dbAddress string) (*SubStore, error) {

  existing, exists := s.getSubstore(dbAddress);
  if exists {
    return &existing, nil
  } else {
    sub, err := s.openKeyValueStore(dbAddress)
    if err != nil {
      return nil, err
    }
    substore, err := s.NewSubStore(sub, false)
    if err != nil {
      return nil, err
    }

    s.setSubstore(dbAddress, substore)
    return substore, nil
  }
}

func (s *Store) subscribeToSubstore(ctx context.Context, db orbitdb.KeyValueStore) () {
  sub := db.Subscribe(ctx)

  if s.Role == ReplicatorRole {
    go func() {
      for e := range sub {
        switch e.(type) {
        // stores.EventReady, stores.EventWrite
        case *stores.EventLoad:
          event := e.(*stores.EventLoad)
          for _, head := range event.Heads {
            err := s.HandleEventPinning(ctx, head)
            if err != nil {
              s.Logger.Debug(err.Error())
            }
          }
        case *stores.EventReplicateProgress:
          event := e.(*stores.EventReplicateProgress)
          err := s.HandleEventPinning(ctx, event.Entry)
          if err != nil {
            s.Logger.Debug(err.Error())
          }
        case *stores.EventReplicated:
          s.Logger.Debug("replicated " + db.Address().String())
          _, err := basestore.SaveSnapshot(ctx, db)
          if err != nil {
            s.Logger.Debug("could not save snapshot")
            s.Logger.Debug(err.Error())
          }
        }
      }
    }()
  }
}

// todo fix cancel and context
func (s *Store) openKeyValueStore(dbAddress string) (orbitdb.KeyValueStore, error) {
      s.Logger.Info("opening keyvaluestore at " + dbAddress)
      var nilKeyValStore orbitdb.KeyValueStore
      isInvalidAddress := address.IsValid(dbAddress)
      if isInvalidAddress == nil {
        ctx, _ := context.WithCancel(context.Background())

        var replication = true
        var create = false
        var storeType = "keyvalue"

        store, err := s.Orbit.Open(ctx, dbAddress, &iface.CreateDBOptions{
            Replicate: &replication,
            Identity: s.Orbit.Identity(),
            Create: &create,
            StoreType: &storeType,
        })
        if err != nil {
          return nilKeyValStore, err
        }
        db, ok := store.(orbitdb.KeyValueStore)
      	if !ok {
      		return nil, errors.New("unable to cast store to keyvalue")
      	}
        if s.Role == ReplicatorRole {
          err = db.LoadFromSnapshot(ctx)
          if err != nil {
            s.Logger.Debug("could not load from snapshot")
          }
        }
        s.Logger.Debug("loading" + dbAddress)
        err = db.Load(ctx, -1)
        if err != nil {
          s.Logger.Debug(err.Error())
        }
        return db, nil
      } else {
        return nilKeyValStore, isInvalidAddress
      }
}

func (s *Store) OpenDb(address string) (iface.KeyValueStore, error) {
  ctx, _ := context.WithCancel(context.Background())
  var replication = true
  var create = false
  var storeType = "keyvalue"
  store, err := s.Orbit.Open(ctx, address, &iface.CreateDBOptions{
    Replicate: &replication,
    Identity: s.Orbit.Identity(),
    Create: &create,
    StoreType: &storeType,
  })
  if err != nil {
    return nil, err
  }
  db, ok := store.(orbitdb.KeyValueStore)
  if !ok {
    return nil, errors.New("unable to cast store to keyvalue")
  }
  return db, nil
}

func (s *Store) CreateOrbit() (error) {

  var orbitIdentity *identityprovider.Identity
  var privKey *crypto.PrivKey
  if s.Role == WorkerRole {
    var createIdentErr error
    orbitIdentity, privKey, createIdentErr = orbit.CreateOrbitIdentity()
    if createIdentErr != nil {
        return createIdentErr
    }
    s.PrivateKey = *privKey
  }

  ctx, _ := context.WithCancel(context.Background())

  //  else {
  //   s.OrbitCache = cacheleveldown.New(&cache.Options{Logger: s.Logger})
  // }

  // var importedTar = make(map[string]string)
  // if s.Config.EnableSnapshots {
  //   seedTar := os.Getenv("SEED_TAR")
  //   if s.IsFirstRepoInit {
  //     if len(seedTar) == 0 {
  //       s.Logger.Debug("No SEED_TAR provided, not attempting to preload stores")
  //     } else {
  //       var importErr error
  //       importedTar, importErr = s.ImportTar(seedTar)
  //       if importErr != nil {
  //         return importErr
  //       }
  //     }
  //   } else if len(seedTar) != 0 {
  //     s.Logger.Debug("SEED_TAR provided, but repo already initialized, skipping preload")
  //   }
  // }

  // TODO check if snapshots enabled, if not make dir memory
  var orbitDirectory *string

  if s.Role == LeecherRole {
    orbitDirectory = &cacheleveldown.InMemoryDirectory
  } else {
    IPFS_DIR := util.GetEnv("IPFS_DIR", "/data/ipfs")
    orbitKeyStore := filepath.Join(IPFS_DIR, "orbitKeystore")
    orbitDirectory = &orbitKeyStore
  }

  orbit, err := orbitdb.NewOrbitDB(ctx, s.Api, &orbitdb.NewOrbitDBOptions{
    Directory: orbitDirectory,
    Identity: orbitIdentity,
    Logger: s.Logger,
    // Cache: s.OrbitCache,
    })
  if err != nil {
      return err
  }
  s.Orbit = orbit
  // preloadErr := s.preloadCids(importedTar)
  // if preloadErr != nil {
  //   panic(preloadErr)
  // }
  return nil
}

func (s *Store) OpenMain() (error) {
    ORBIT_DB_ADDRESS := util.GetEnv("ORBIT_DB_ADDRESS", "")
    addressError := address.IsValid(ORBIT_DB_ADDRESS)
    if len(ORBIT_DB_ADDRESS) == 0 {
        return errors.New("ORBIT_DB_ADDRESS is not set!")
    } else if addressError != nil {
        return errors.New("ORBIT_DB_ADDRESS is not valid!")
    }
    db, err := s.openKeyValueStore(ORBIT_DB_ADDRESS)
    if err != nil {
      return err
    }
    s.main = db
    return nil
}

func NewStore(o *CreateStoreOptions) (Store, error) {
  var s Store
  s.SubStores = make(map[string]SubStore)
  s.SubStoresLock = sync.RWMutex{}
  s.SubStoreOf = make(map[string]string)
  s.SubStoreOfLock = sync.RWMutex{}
  s.ChannelSuffix = string("_" + string(rand.Intn(ChannelShards)))

  // Configure Store
  opt := *o
  if opt.Logger == nil {
    s.Logger = zap.NewExample()
    defer s.Logger.Sync()
  }
  s.Role = opt.Role

  // Configure Replicator-specific options
  if opt.Role == ReplicatorRole {
    if opt.FractionPin <= 0 {
      opt.FractionPin = 1
    }
    if opt.MaxGb <= 0 {
      opt.MaxGb = 10
    }
    if opt.EnableCache {
      err := s.CreateCache()
      if err != nil {
        return s, err
      }
    }
    if opt.ManageWorkers {
      s.AsyncClient = asynq.NewClient(tasks.NewAsyncRedisConnection())
      // todo break here if cannot connect
      // defer asyncClient.Close()
    }
  }

  err := s.createApi()
  if !opt.SkipOrbit {
    err := s.CreateOrbit()
    if err != nil {
      return s, err
    }
    err = s.OpenMain()
    if err != nil {
      return s, err
    }
    err = s.configureMain()
    if err != nil {
      return s, err
    }
  }

  return s, err
}
