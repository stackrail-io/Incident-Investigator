# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/incident-investigator ./cmd/incident-investigator

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/incident-investigator /usr/local/bin/incident-investigator

# MCP speaks over stdio; logs go to stderr.
ENTRYPOINT ["incident-investigator"]
