FROM golang:1.18 AS builder

RUN apt-get update && apt-get install --yes libsodium-dev

COPY ./ /go/src/github.com/zostay/garotate/

WORKDIR /go/src/github.com/zostay/garotate

RUN make clean && make test && make install

FROM alpine AS application

COPY --from=builder /go/bin/garotate /usr/local/bin/garotate

CMD ["/usr/local/bin/garotate"]
