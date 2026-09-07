# Build Geth in a stock Go builder container
FROM golang:1.18-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /go-ethereum
RUN cd /go-ethereum && make geth

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
RUN mkdir "/viction"
COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/
COPY ./entrypoint.sh /viction/entrypoint.sh
RUN chmod +x /viction/entrypoint.sh

# Configuration defaults, overridable at runtime with `docker run -e`
ENV NETWORK="viction"
ENV NETWORK_ID="88"
ENV IDENTITY="mynode"
ENV HTTP_API="eth,web3"
ENV WS_API="eth,web3"
ENV P2P_PORT="30303"
ENV EXTIP=""
ENV MAXPEERS="100"
ENV SYNCMODE="full"
ENV VERBOSITY="3"
ENV GCMODE="full"
ENV ETHSTATS_HOST=""
ENV ETHSTATS_PORT=""
ENV ETHSTATS_SECRET=""
ENV PRIVKEY=""
ENV PASSWORD=""

EXPOSE 8545 8546 ${P2P_PORT} ${P2P_PORT}/udp
ENTRYPOINT ["/viction/entrypoint.sh"]
