FROM golang:1.18.0-alpine3.15 AS builder

RUN apk update && \
    apk add --no-cache git

WORKDIR $GOPATH/src/tim-hat-die-hand-an-der-maus/plex-resolver
COPY . .

RUN go get -d -v
RUN CGOENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s -extldflags=-static" -o /go/bin/plex-resolver
RUN ldd /go/bin/plex-resolver

FROM scratch

COPY --from=builder /go/bin/plex-resolver /go/bin/plex-resolver
COPY --from=builder /lib/ld-musl-x86_64.so.1 /lib/ld-musl-x86_64.so.1
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/go/bin/plex-resolver"]
