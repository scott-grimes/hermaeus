package conf

import (
	"os"
  "fmt"
  "errors"
  "berty.tech/go-ipfs-log/identityprovider"
  ks "berty.tech/go-ipfs-log/keystore"
	"github.com/ipfs/go-datastore"
  "berty.tech/go-orbit-db/address"
	"encoding/base64"

)
type Configuration struct {
	PrivateKeyBytes   []byte
	OrbitDbAddress    string
  Identity *identityprovider.Identity
  Keystore *ks.Keystore
  KeystoreInternal datastore.Datastore
}

func Build() (Configuration, error) {

	var conf Configuration
  // todo make list of things
  IPFS_IDENTITY := os.Getenv("IPFS_IDENTITY")
  if len(IPFS_IDENTITY) == 0 {
			return conf, errors.New("IPFS_IDENTITY must be set!")
  }

  ORBIT_DB_ADDRESS := os.Getenv("ORBIT_DB_ADDRESS")
  addressError := address.IsValid(ORBIT_DB_ADDRESS)
  if len(ORBIT_DB_ADDRESS) == 0 {
      fmt.Println("Warn: ORBIT_DB_ADDRESS is not set. Attempting to open a DB will fail")
  } else if addressError != nil {
    return conf, errors.New("ORBIT_DB_ADDRESS is not valid!")
  }

  privKeyBytes, err := base64.StdEncoding.DecodeString(IPFS_IDENTITY)
  if err != nil {
		return conf, err
	}

	conf.PrivateKeyBytes = privKeyBytes
	conf.OrbitDbAddress = ORBIT_DB_ADDRESS
  return conf, nil

}
