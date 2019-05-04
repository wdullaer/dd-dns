FROM golang:1.12-alpine AS builder

WORKDIR /app/dd-dns
RUN apk --no-cache add git

# Download the dependencies first, they don't change often
# Doing this improves caching and build times
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy remaining files and build the binary
COPY . .
RUN go build

FROM alpine:latest

CMD ["/dd-dns", "--data-directory", "/data"]
VOLUME /data
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/dd-dns/dd-dns /dd-dns
