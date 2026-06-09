FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
RUN git clone --depth=1 https://github.com/tamnd/liteio.git .
RUN go build -o /liteio ./cmd/liteio

FROM alpine:3.21
RUN apk add --no-cache curl && mkdir -p /data/d1 /data/d2 /data/d3 /data/d4
COPY --from=builder /liteio /usr/local/bin/liteio
EXPOSE 9000 9001
ENTRYPOINT ["liteio"]
CMD ["--drives", "/data/d1,/data/d2,/data/d3,/data/d4", "--parity", "1"]
