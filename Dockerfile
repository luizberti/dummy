FROM golang:alpine AS builder
WORKDIR /go/src/build/

COPY *.go .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-s -w' -gcflags=-trimpath=x/y -o main *.go


FROM alpine:latest AS final
COPY --from=builder /go/src/build/main /tmp/
RUN apk add --no-cache ca-certificates

USER nobody
EXPOSE 5000
ENTRYPOINT ["/tmp/main"]

