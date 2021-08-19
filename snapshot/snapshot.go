package snapshot

import (
    "fmt"
    "github.com/ipfs/go-cid"
    "bytes"
    "context"
    "archive/tar"
    "io/ioutil"
    "time"
    "encoding/base64"
    "io"
    "github.com/ipfs/go-ipfs-files"
    ipfspath "github.com/ipfs/interface-go-ipfs-core/path"
    corepath "github.com/ipfs/interface-go-ipfs-core/path"
)

type TarSubStore struct {
  Address string
  Cid cid.Cid
  Snapshot cid.Cid
}

func (s *Store) getFileBytes(c cid.Cid) ([]byte, error){
  path := corepath.IpfsPath(c)
  file, err := s.Api.Unixfs().Get(context.TODO(), path)
	if err != nil {
		return nil, err
	}
  fb, err := ioutil.ReadAll(files.ToFile(file))
  if err != nil {
		return nil, err
	}
  return fb, nil
}

func (s *Store) ExportTar(substores []TarSubStore) (error) {
  var buf bytes.Buffer

	tw := tar.NewWriter(&buf)

	for _, substore := range substores {
    // get snapshot
    // f, err := s.GetDocument(context.TODO(), substore.Snapshot.String())
    // fb, err := ioutil.ReadAll(f)
    // if err != nil {
  	// 	return err
  	// }

    fb, err := s.getFileBytes(substore.Snapshot)
    if err != nil {
  		return err
  	}

    name := base64.URLEncoding.EncodeToString([]byte("snapshot"+substore.Address))
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(string(fb))),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(fb); err != nil {
			return err
		}

    // f, err = s.GetDocument(context.TODO(), substore.Cid.String())
    // if err != nil {
  	// 	return err
  	// }
    // fb, err = ioutil.ReadAll(f)
    // if err != nil {
  	// 	return err
  	// }
    fb, err = s.getFileBytes(substore.Snapshot)
    if err != nil {
      return err
    }
    name = base64.URLEncoding.EncodeToString([]byte("cid"+substore.Address))
		hdr = &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(string(fb))),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(fb); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
  now := fmt.Sprint(time.Now().Unix())

  err := ioutil.WriteFile("./hermaeus_snapshot_"+ now + ".tar", buf.Bytes(), 0644)
  if err != nil {
    return err
  }
  return nil
}

func (s *Store) ImportTar(seedTar string) (map[string]string, error) {
  var importedTar = make(map[string]string)

  dat, err := ioutil.ReadFile(seedTar)
  if err != nil {
    return importedTar, err
  }
  buf := bytes.NewBuffer(dat)

  tr := tar.NewReader(buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
      // End of archive
			break
		}
		if err != nil {
			return importedTar, err
		}

    b, err := io.ReadAll(tr)
		if err != nil {
			return importedTar, err
		}
    bytesFile := files.NewBytesFile(b)
    var p ipfspath.Resolved
    p, err = s.Api.Unixfs().Add(context.TODO(), bytesFile)
  	if err != nil {
  		return importedTar, err
  	}
    name, _ := base64.StdEncoding.DecodeString(hdr.Name)
		s.Logger.Debug("added to ipfs " + string(name) + " with value " + p.Root().String())

    // croot := p.Root().String()
    //importedTar[string(name)] = croot

  }

  return importedTar, nil
}
