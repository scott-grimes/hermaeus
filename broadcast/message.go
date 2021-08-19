package broadcast

import (
  orbitdb "berty.tech/go-orbit-db"
  "fmt"
  "encoding/json"
  "errors"
  // "go.uber.org/zap"
  // "context"
  "example.com/m/v2/entry"
  "github.com/libp2p/go-libp2p-core/crypto"
  "encoding/base64"
  "encoding/hex"

)

var UnverifiedAuthor = "unverified"

// todo make this store, dedupe
const KeyLength = 4

type Message struct {
  Type string           `json:"type"`
  Payload string        `json:"payload"`
}

type RequestDoc struct {
	DocId string             `json:"id"`
}

// Response is Entry
//
// // set when a peer requests something we do not have
// ignorePeer := cache.New(5*time.Minute, 10*time.Minute)
//
// // inc. used to block spammy peers. should expire after certain number?
// peerRequestCount := cache.New(5*time.Minute, 10*time.Minute)


func messageWasAuthorized(author string, s orbitdb.KeyValueStore) (bool, error) {
  allowedUsers, err := s.AccessController().GetAuthorizedByRole("write")
  if err != nil {
    return false, err
  }
  for _, user := range allowedUsers {
    if user == author {
      fmt.Println("authorized found!")
      return true, nil
    }
  }
  return false, nil
}

//todo right now we only verify the first author. need to validate all
func fetchVerifiedAuthorResponseDoc(msg Message) (string, error) {
	var entries []entry.Entry
	err := json.Unmarshal([]byte(msg.Payload), &entries);
	if err != nil {
		return UnverifiedAuthor, err
	}
  e := entries[0]
	signedPortion := entry.Entry{
		DocId: e.DocId,
		IpfsRef: e.IpfsRef,
    Content: e.Content,
		Author: e.Author,
	}
	payload, payloadErr := json.Marshal(signedPortion)
	if payloadErr != nil {
		return UnverifiedAuthor, payloadErr
	}
	ok, err := VerifySignature(payload, e.Signature, e.Author)
  if err != nil {
    return UnverifiedAuthor, err
  }
  if !ok {
    return UnverifiedAuthor, errors.New("Failed signature validation")
  }
	return e.Author, nil
}

func ValidateMessage(msg Message, root orbitdb.KeyValueStore) (bool, error) {

	var author string
	var verificationError error
  if msg.Type == "RequestDoc" {
    return true, nil
  } else if msg.Type == "ResponseDoc" {
		author, verificationError = fetchVerifiedAuthorResponseDoc(msg)
	} else {
    return false, errors.New("Unknown message type encountered during validation")
  }
	if verificationError != nil {
		return false, verificationError
	}
	if author == UnverifiedAuthor {
		return false, errors.New("Validation of Message Failed")
	}
  fmt.Println("before entering messageWasAuthorized",author,root)
  ok, err := messageWasAuthorized(author, root)
  if err != nil {
    return false, err
  }
  if !ok {
    return false, nil
  }
  return true, nil
}


func SignMessage(msg string, privkey crypto.PrivKey) (string, error) {
  res, err := privkey.Sign([]byte(msg))
  if err != nil {
    return "", err
  }
  return base64.StdEncoding.EncodeToString(res), err
}

func VerifySignature(msg []byte, b64Sig string, authorHex string) (bool, error) {
  pubKeyBytes, err := hex.DecodeString(authorHex)
  if err != nil {
    return false, err
  }
  pubKey, err := crypto.UnmarshalSecp256k1PublicKey(pubKeyBytes)
  if err != nil {
    return false, err
  }
  sig, err := base64.StdEncoding.DecodeString(b64Sig)
  if err != nil {
    return false, err
  }
  ok, err := pubKey.Verify(msg,[]byte(sig))
  if err != nil {
    return false, err
  }
  return ok, nil
}
