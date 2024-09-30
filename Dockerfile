FROM golang:1.20

WORKDIR /root/app

# COPY client client/
# COPY delayProxy delayProxy/
# COPY routingProxy routingProxy/
# COPY server server/
# COPY util util/
COPY rope-go rope-go

ARG GIT_COMMIT="dev-pc"
ENV GIT_COMMIT=$GIT_COMMIT

#Compiling client
WORKDIR /root/app/rope-go/client
RUN go mod download; go build -ldflags "-w -s -X 'main.GitCommit=$GIT_COMMIT'" -tags 'osusergo netgo' -o /build/client client.go fixed.go forwardingConf.go

#Compiling routing
WORKDIR /root/app/rope-go/routing
RUN go mod download; go build -ldflags "-w -s -X 'main.GitCommit=$GIT_COMMIT'" -tags 'osusergo netgo' -o /build/routing proxy.go probability.go probabilityLatency.go fixed.go forwardingConf.go

#Compiling server
WORKDIR /root/app/rope-go/server
RUN go mod download; go build -ldflags "-w -s -X 'main.GitCommit=$GIT_COMMIT'" -tags 'osusergo netgo' -o /build/server server.go reply.go replyrelay.go forwardingConf.go

FROM scratch

EXPOSE 4040/udp

ARG binary

COPY --from=0 /build/$binary /app

ENTRYPOINT [ "/app" ]
