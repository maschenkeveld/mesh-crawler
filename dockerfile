# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o server .

FROM alpine:3.18

RUN apk add --no-cache ca-certificates curl openssl

WORKDIR /app
COPY --from=builder /app/server /app/server

CMD ["/app/server"]