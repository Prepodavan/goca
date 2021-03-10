FROM golang:alpine AS builder

RUN apk add --no-cache openssl

WORKDIR /src

COPY go.mod go.mod
RUN go mod download

COPY cmd cmd
COPY internal internal
COPY pkg pkg

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w -s" -o /app/server ./cmd/issuer

COPY api api
COPY assets assets

EXPOSE 8080
ENV GIN_MODE=release

ENTRYPOINT ["/app/server", "-address", "0.0.0.0:8080"]
