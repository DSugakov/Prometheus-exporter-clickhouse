# Build
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=git" -o /ch-exporter ./cmd/ch-exporter

# Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /ch-exporter /usr/local/bin/ch-exporter
EXPOSE 9101
USER nobody
ENTRYPOINT ["/usr/local/bin/ch-exporter"]
