FROM golang:1.10-alpine as builder
RUN apk add -u git
WORKDIR /go/src/github.com/google/huproxy
COPY . ./
RUN go get ./...
RUN \
    env GOOS=linux GARCH=amd64 CGO_ENABLED=0 go build -o /opt/huproxy huproxy.go && \
    env GOOS=linux GARCH=amd64 CGO_ENABLED=0 go build -o /opt/huproxyclient_linux_amd64 huproxyclient/client.go && \
    env GOOS=darwin GARCH=amd64 CGO_ENABLED=0 go build -o /opt/huproxyclient_darwin_amd64 huproxyclient/client.go && \
    env GOOS=windows GARCH=amd64 CGO_ENABLED=0 go build -o /opt/huproxyclient_win64.exe huproxyclient/client.go
