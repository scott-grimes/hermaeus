package main

// Prints base64 encoded ipfs identity (private Secp256k1 key)
//
// Usage
// go run createIdentity.go
//

import (
    "fmt"
    "github.com/libp2p/go-libp2p-core/crypto"
    "encoding/base64"
    "crypto/rand"
    "encoding/hex"
)

func main() {
  priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		fmt.Println("bad key")
	}
	keyBytes, err := priv.Raw()

  ident := base64.StdEncoding.EncodeToString(keyBytes)
  pubKey, err := priv.GetPublic().Raw()
  pubKeyHex := hex.EncodeToString(pubKey)
  fmt.Println("{\"private\":\"" + ident + "\",\"public\":\"" + pubKeyHex + "\"}")

}
