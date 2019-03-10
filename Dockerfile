FROM golang:1.12-alpine AS builder

WORKDIR /app/dd-dns
RUN apk --update add git
COPY . .

RUN go build

FROM alpine:latest

CMD ["/dd-dns", "--data-directory", "/data"]
VOLUME /data
COPY --from=builder /app/dd-dns/docker-dns-updater /dd-dns
