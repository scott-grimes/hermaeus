# Hermaeus

A decentralized p2p document store running on top of ipfs.

A fun weekend project / POC

Example usecase - storing and retrieving pdf documents

Ie. Enter an ISBN, DOI, PMID, or arXiv: return a pdf

It works as is but, being my first go project, would require significant work to harden and make production ready

# Architecture
*Replicators*: Store the database itself along with documents. Anyone (yes you) can run a replicator!

*Workers*: Scalable workers respond to requests over shared channel by adding new documents to the database which are replicated. Workers can optionally include metadata from any external resource along with the document. Only authenticated individuals can write to the database, or store documents

*Leechers*: Can request documents and metadata. Anyone can leech! Requests are sent via ipfs pubsub rooms, or by http requests.

# Info
Anyone can run a replicator, configuring it to store some fraction of the database up to a pre-defined storage limit.

Retrieval is fast(ish) - db supports 16-65536 possible shards (16 default)

More importantly, documents and the database itself are distributed - once instantiated it is very resilient to being taken offline

Use of private keys ensures gives granular control over who has write access to database. Privileges can be revoked.

Rudimentary anti-spam measures in place to prevent flooding users with bad requests. Documents (and the associated metadata) are signed, making it impossible for bad actors to ship bogus data to requesters

### Setup

1) Create a new orbitdb identity. This will be the owner of the database

```
go run bin/createIdentity.go | jq -r '.private'
```

2) Instantiate a new database

```
export ORBIT_DB_IDENTITY=$(go run createIdentity.go | jq -r '.private')
go run create/createDatabase.go
```

3) Spin up replicators. The database is now persisted on ipfs
```
echo ORBIT_DB_ADDRESS=$ORBIT_DB_ADDRESS > .env
docker-compose up -d
```

4) Basic front-end website is included, you can also run requestdoc and putdoc from `post/` directly to fetch and set documents by id.
