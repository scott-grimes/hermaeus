package store

import (
  // "berty.tech/go-ipfs-log/identityprovider"
  // "berty.tech/go-orbit-db/address"
  // "berty.tech/go-orbit-db/iface"
  // "berty.tech/go-orbit-db/stores"
  // "berty.tech/go-orbit-db/stores/basestore"
  "bytes"
  // "github.com/hibiken/asynq"
  // "example.com/m/v2/worker/tasks"
  "context"
  "github.com/ipfs/go-ipfs-files"
  "io"
  "io/ioutil"
  // "math/rand"
  "berty.tech/go-orbit-db/stores/operation"
  // "encoding/base64"
  "errors"
  "example.com/m/v2/util"
  corepath "github.com/ipfs/interface-go-ipfs-core/path"
  ipfslog "berty.tech/go-ipfs-log"
  "fmt"
  // "berty.tech/go-ipfs-log/keystore"
  // "reflect"
  // "unsafe"
  // "berty.tech/go-orbit-db/cache"
  // "github.com/go-redis/redis/v8"
  // "berty.tech/go-orbit-db/cache/cacheleveldown"
  // "github.com/ipfs/go-datastore"
  // "github.com/libp2p/go-libp2p-core/crypto"
  // "encoding/json"
  "github.com/ipfs/go-cid"
  "github.com/ipfs/go-ipfs-config"
  "github.com/ipfs/go-ipfs/core/coreapi"
  "github.com/ipfs/go-ipfs/core/node/libp2p"
  "github.com/ipfs/go-ipfs/plugin/loader"
  // coreinterface "github.com/ipfs/interface-go-ipfs-core"
  repo "github.com/ipfs/go-ipfs/repo"
  "github.com/ipfs/interface-go-ipfs-core/options"
  // "go.uber.org/zap"
  "path/filepath"
  // "sync"
  // "strings"
  // "time"
  // "os"
  core "github.com/ipfs/go-ipfs/core"
  fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"
  // orbitdb "berty.tech/go-orbit-db"
)

func (s *Store) createIpfsRepo() (*repo.Repo, error) {
  IPFS_DIR := util.GetEnv("IPFS_DIR", "/data/ipfs")
  API_PORT := util.GetEnv("API_PORT", "5001")
  SWARM_PORT := util.GetEnv("SWARM_PORT", "4001")

  writer := bytes.NewBuffer(nil)
  identity, err := config.CreateIdentity(writer, []options.KeyGenerateOption{options.Key.Type(options.Ed25519Key)})
  repoCfg, err := config.InitWithIdentity(identity)
  if err != nil {
    return nil, err
  }
  repoCfg.Addresses.API = config.Strings{"/ip4/127.0.0.1/tcp/" + API_PORT}
  repoCfg.Addresses.Swarm = []string{
      "/ip4/0.0.0.0/tcp/" + SWARM_PORT,
      "/ip6/::/tcp/" + SWARM_PORT,
      "/ip4/0.0.0.0/udp/" + SWARM_PORT + "/quic",
      "/ip6/::/udp/" + SWARM_PORT + "/quic",
      }

  // fsrepo requires plugins
  fmt.Println(filepath.Join(IPFS_DIR, "plugins"))
  plugins, err := loader.NewPluginLoader(filepath.Join(IPFS_DIR, "plugins"))
  if err != nil {
    return nil, err
  }
  if err := plugins.Initialize(); err != nil {
    return nil, err
  }
  if err := plugins.Inject(); err != nil {
    return nil, err
  }
  // init ipfs on first run
  isInitalized := fsrepo.IsInitialized(IPFS_DIR)
  if isInitalized != true {
    if err := fsrepo.Init(IPFS_DIR, repoCfg); err != nil {
        return nil, err
      }
    s.IsFirstRepoInit = true
  }

  repo, err := fsrepo.Open(IPFS_DIR)
  if err != nil {
    return nil, err
  }
  return &repo, nil
}

func (s *Store) createApi() error {
  nodeCfg := &core.BuildCfg{
    Routing: libp2p.DHTServerOption, // will query other peers for DHT records, and will respond to requests from other peers (both requests to store records and requests to retrieve records
    Online: true,
    ExtraOpts: map[string]bool{
      "pubsub": true,
    },
  }

  // todo, if a leecher role we still wish to accept api
  if s.Role != LeecherRole {
    repo, repoErr := s.createIpfsRepo()
    if repoErr != nil {
      return repoErr
    }
    nodeCfg.Repo = *repo
  }

  nodeCtx, _ := context.WithCancel(context.Background())

  node, err := core.NewNode(nodeCtx, nodeCfg)
  if err != nil {
    s.Logger.Panic(err.Error())
  }
  s.Node = node
  api, err := coreapi.NewCoreAPI(node)
  if err != nil {
    return err
  }
  s.Api = api
  s.Logger.Debug("***********************************")
  s.Logger.Debug("IPFS INFO")
  s.Logger.Debug("node id:")
  s.Logger.Debug(fmt.Sprint(s.Node.PeerHost.ID()))
  s.Logger.Debug("addresses:")
  s.Logger.Debug(fmt.Sprint(s.Node.PeerHost.Addrs()))
  s.Logger.Debug("***********************************")
  return nil
}

// Saves file to ipfs, returns ipfsRef
// does not pin file
func (s *Store) SaveDocument(ctx context.Context, file io.Reader) (string, error) {
  data, err := ioutil.ReadAll(file)
    if err != nil {
      return "", nil
    }
  n := files.NewBytesFile(data)
  obj, err := s.Api.Unixfs().Add(ctx, n)
	if err != nil {
		return "", err
	}
  return obj.Root().String(), nil
}

func (s *Store) GetDocument(ctx context.Context, ipfsRef string) (files.File, error) {
  c, err := cid.Decode(ipfsRef)
  if err != nil {
    return nil, err
  }
  path := corepath.IpfsPath(c)
  file, err := s.Api.Unixfs().Get(ctx, path)
	if err != nil {
		return nil, err
	}
  return files.ToFile(file), nil
}


// todo determnistic randomness on percent pinned
func (s *Store) HandleEventPinning(ctx context.Context, event ipfslog.Entry) error {
  op, err := operation.ParseOperation(event)
  if err != nil {
    return errors.New("operation parse failed for ")
  } else {
    opType := op.GetOperation()
    ipfsRef := string(op.GetValue())
    if opType == "PUT" {
      err = s.Pin(ctx, ipfsRef)
      if err != nil {
        return err
      } else {
        s.Logger.Debug("pinned " + ipfsRef)
      }
    } else if opType == "DEL" {
      s.Logger.Debug("no unpinning handled yet")
    } else {
      return errors.New("unknown operation type: " + opType)
    }
  }
  return nil
}

// pins a given ipfsRef
// todo impliment only for keys on root nodes
func (s *Store) Pin(ctx context.Context, ipfsRef string) error {
  c, err := cid.Decode(ipfsRef)
  if err != nil {
    return err
  }
  path := corepath.IpfsPath(c)
  return s.Api.Pin().Add(ctx, path)
}
