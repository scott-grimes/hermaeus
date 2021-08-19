package main

import (
    "fmt"
    "context"
    "time"
    // "berty.tech/go-orbit-db/stores/basestore"
    "bytes"
    "berty.tech/go-orbit-db/iface"
    "berty.tech/go-orbit-db/accesscontroller"
    "encoding/json"
    "example.com/m/v2/store"
    // "example.com/m/v2/snapshot"
)

func main() {
  ctx := context.TODO()
  s, err := store.NewStore(&store.CreateStoreOptions{
    Role: store.WorkerRole,
    SkipOrbit: true,
  })

  err = s.CreateOrbit()
  if err != nil {
   panic(err)
  }

  var replication = true
  db, err := s.Orbit.KeyValue(ctx, "hermaeus", &iface.CreateDBOptions{
      Identity: s.Orbit.Identity(),
      Replicate: &replication,
   })
  if err != nil {
   panic(err)
  }

  someerr := db.Load(ctx,-1)
  if someerr != nil {
    panic(err)
  }
  // Instantiates database with single layer
  // Main Hermaeus Database {"root": BranchAddress}
  // Branch Database {"keylength" : "int", "k:<>": subAddress}
  // Leaf Database {"key": ipfsRef }
  // todo deprecate leaf in favor of checking keylength

  now := fmt.Sprint(time.Now().Unix())
  // now := "1624844936"

  rootDb, _ := s.Orbit.KeyValue(ctx, "root_" + now, &iface.CreateDBOptions{
      Identity: s.Orbit.Identity(),
      Replicate: &replication,
   })
   _, putErr2 := db.Put(ctx, "root", []byte(rootDb.Address().String()) )
   if putErr2 != nil {
     panic(putErr2)
   }

   yearFromNow := fmt.Sprintf("%d",time.Now().AddDate(1, 0, 0).Unix())
   secondWriter := "037ec8dcbf852a3ac48613017d963a762e0b891d0bf762ed148c5904116548941c"

   authorized := make(map[string]string)
   authorized[rootDb.Identity().ID] = fmt.Sprintf("%d",time.Now().AddDate(666, 0, 0).Unix())
   authorized[secondWriter] = yearFromNow

   auth, autherr := json.Marshal(authorized)
   if autherr != nil {
     panic(autherr)
   }

   _, authPutErr := rootDb.Put(ctx, "authorized", auth )
   if authPutErr != nil {
     panic(authPutErr)
   }

  _, verErr := rootDb.Put(ctx, "version", []byte("0.0.0") )
  if verErr != nil {
    panic(verErr)
  }

  subStoreAc := &accesscontroller.CreateAccessControllerOptions{
       Access: map[string][]string{
         "write": {
           rootDb.Identity().ID,
           secondWriter,
         },
       },
     }

  subStoreAddresses := make(map[string]string)
  // var exportList []snapshot.TarSubStore

  for i := 0; i <16; i++ {
    key := fmt.Sprintf("%x", i)
    tempDb, err := s.Orbit.KeyValue(ctx, key + "xxx_" + now, &iface.CreateDBOptions{
        Identity: s.Orbit.Identity(),
        Replicate: &replication,
        AccessController: subStoreAc,
     })
     if err != nil {
       panic(err)
     }
    tempDb.Load(context.TODO(),-1)
    _, putErr3 := tempDb.Put(ctx, "prefix", []byte(key))
    if putErr3 != nil {
      panic(putErr3)
    }
    // c, err := basestore.SaveSnapshot(ctx, tempDb)
    // if err != nil {
    //   panic(err)
    // }
    // exportList = append(exportList,snapshot.TarSubStore{
    //   Address: tempDb.Address().String(),
    //   Snapshot: c,
    //   Cid: tempDb.Address().GetRoot(),
    //   })

    for j := 0; j <16; j++ {
      for k := 0; k <16; k++ {
        for l := 0; l <16; l++ {
          key2 := fmt.Sprintf("%x%x%x%x", i, j, k, l)
          subStoreAddresses[key2] = tempDb.Address().String()
        }
      }
    }
  }

   substoresJsonBytes, substoreserr := json.Marshal(subStoreAddresses)
   if substoreserr != nil {
     panic(substoreserr)
   }

  substoresJsonReader := bytes.NewReader(substoresJsonBytes)
  substoresJsonIpfsRef, err := s.SaveDocument(context.TODO(),substoresJsonReader)

  _, putErr := rootDb.Put(ctx, "substores", []byte(substoresJsonIpfsRef) )
  if putErr != nil {
    panic(putErr)
  }

  _, putErr = rootDb.Put(ctx, "bannermessage", []byte("Banner Message Here") )
  if putErr != nil {
    panic(putErr)
  }

  fmt.Println(db.Address().String())

  // basePath, _ := basestore.SaveSnapshot(ctx, db)
  // exportList = append(exportList,snapshot.TarSubStore{
  //   Address: db.Address().String(),
  //   Snapshot: basePath,
  //   Cid: db.Address().GetRoot(),
  //   })
  //// blerg, _ := db.Cache().Get(datastore.NewKey("snapshot"))
  //// fmt.Println("snapshot from cache")
  //// fmt.Println(string(blerg))
  //
  // rootPath, _ := basestore.SaveSnapshot(ctx, rootDb)
  // exportList = append(exportList,snapshot.TarSubStore{
  //   Address: rootDb.Address().String(),
  //   Snapshot: rootPath,
  //   Cid: rootDb.Address().GetRoot(),
  // })
  //
  // err = s.ExportTar(exportList)
  // if err != nil {
  //   panic(err)
  // }

  // for k,v := range rootDb.All() {
  //   fmt.Println(k,string(v))
  // }
  // importTar("./hermaeus_snapshot.tar")
  time.Sleep(time.Duration(1<<63 - 1))

  defer db.Close()

}
