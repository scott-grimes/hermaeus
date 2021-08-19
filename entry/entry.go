package entry

import (
  "github.com/libp2p/go-libp2p-core/crypto"
  "encoding/json"
  "errors"
  "encoding/base64"
  "encoding/hex"
)

// todo make entries multiple to handle hash collisions
type Entry struct {
  DocId string `json:"id"`
  // location of the document
  IpfsRef string `json:"ipfs"`
  // json metadata which can be surfaced about the document
  Content string `json:"content"`
  // hex-encoded public key of writer
  Author string `json:"signer"`
  // b64 signature of the author
  Signature string `json:"signature,omitempty"`
}

func SignEntry(entry Entry, privKey crypto.PrivKey) (string, error) {

  partialEntry := Entry{
    DocId: entry.DocId,
    IpfsRef: entry.IpfsRef,
    Content: entry.Content,
    Author: entry.Author,
  }

  b, err := json.Marshal(partialEntry);
  if err != nil {
      return "", err;
  }
  signature, err := privKey.Sign(b)
  if err != nil {
    return "", err;
  }
  b64Sig := base64.StdEncoding.EncodeToString(signature)
  return b64Sig, nil
}

// todo what calls this? needs to support multiple entries
func VerifyEntry(entry Entry) (bool, error) {
  pubKeyBytes, err := hex.DecodeString(entry.Author)
	if err != nil {
		return false, err
	}
  publicKey, err := crypto.UnmarshalSecp256k1PublicKey(pubKeyBytes)
  if err != nil {
    return false, err
  }
  partialEntry := Entry{
    DocId: entry.DocId,
    IpfsRef: entry.IpfsRef,
    Content: entry.Content,
    Author: entry.Author,
  }
  b, err := json.Marshal(partialEntry);
  if err != nil {
      return false, err;
  }
  b64Sig := entry.Signature
  sig, err := base64.StdEncoding.DecodeString(b64Sig)
  if err != nil {
    return false, err
  }
  ok, err := publicKey.Verify(b, []byte(sig))
  if err != nil {
    return false, err
  }
  if !ok {
    return false, errors.New("validation of entry failed")
  }
  return true, nil
}
