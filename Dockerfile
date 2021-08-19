FROM golang
WORKDIR /app
ADD . /app

RUN go get .

ENTRYPOINT ["go", "run", "bin/replicator.go"]
