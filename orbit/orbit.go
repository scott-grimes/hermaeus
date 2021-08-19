package orbit

import (
    "berty.tech/go-ipfs-log/identityprovider"
    "github.com/libp2p/go-libp2p-core/crypto"
    "berty.tech/go-ipfs-log/keystore"
    "github.com/ipfs/go-datastore"
    "encoding/base64"
    "os"
    dssync "github.com/ipfs/go-datastore/sync"
    "reflect"
    "errors"
    "encoding/hex"
    "unsafe"
)

func CreateOrbitIdentity() (*identityprovider.Identity, *crypto.PrivKey, error) {
    var pk crypto.PrivKey
    pkptr, err := getPrivKey()
    if err != nil {
      return nil, nil, err
    }
    pk = *pkptr
    privKeyBytes, err := pk.Raw()
    if err != nil {
      return nil, nil, err
    }
    pubKey, err := pk.GetPublic().Raw()
    if err != nil {
      return nil, nil, err
    }
    IPFS_IDENTITY_NAME := hex.EncodeToString(pubKey)

    ds := dssync.MutexWrap(datastore.NewMapDatastore())
    keystore, err := keystore.NewKeystore(ds)
    if err != nil {
      return nil, nil, err
    }
    keystoreInternal, err := unwrapKeystore(keystore)
    if err != nil {
      return nil, nil, err
    }
    keystoreInternal.Put(datastore.NewKey(IPFS_IDENTITY_NAME), privKeyBytes)
    identity, err := identityprovider.CreateIdentity(&identityprovider.CreateIdentityOptions{
      Keystore: keystore,
      ID:       IPFS_IDENTITY_NAME,
      Type:     "orbitdb",
    })
    if err != nil {
      return nil, nil, err
    }
    return identity, pkptr, err
}


func getPrivKey() (*crypto.PrivKey, error) {

  ORBIT_DB_IDENTITY := os.Getenv("ORBIT_DB_IDENTITY")
  if len(ORBIT_DB_IDENTITY) == 0 {
    return nil, errors.New("ORBIT_DB_IDENTITY must be set!")
  }
  privKeyBytes, err := base64.StdEncoding.DecodeString(ORBIT_DB_IDENTITY)
  if err != nil {
    return nil, err
  }
  privKey, err := crypto.UnmarshalSecp256k1PrivateKey(privKeyBytes)
  if err != nil {
    return nil, err
  }
  return &privKey, nil
}

func unwrapKeystore(oldKeystore *keystore.Keystore) (datastore.Datastore, error) {
  var s datastore.Datastore

  rs := reflect.ValueOf(*oldKeystore)

	rf := rs.Field(0)
	rs1 := reflect.New(rs.Type()).Elem()
	rs1.Set(rs)
	rf = rs1.Field(0)
	rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()

	ri1 := reflect.ValueOf(&s).Elem()
	ri1.Set(rf)

  return s, nil
}
