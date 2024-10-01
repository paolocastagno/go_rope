package main

import (
	"client"
	"github.com/quic-go/quic-go"
	"log"
	"routing"
	"server"
	"time"
)

func main() {
	srv := &server.Server{
		IdDevice:   "Server",
		ListenAddr: "localhost:8080",
		AppCfg:     "../cfg/poa/server/app_reply.toml",
		QueueLen:   10,
		Workers:    5,
		Timeout:    30 * time.Second,
	}

	prx := &routing.Proxy{
		RoutingTbl:   make(map[string]string),
		IdDevice:     "RoutingProxy",
		ListenAddr:   "localhost:8181",
		ConfFile:     "../cfg/poa/routing/cfg.toml",
		Timeout:      30 * time.Second,
		ParallelConn: 0,
	}
	prx.RoutingTbl["localhost:8080"] = "localhost:8080"

	quicConf := &quic.Config{
		MaxIdleTimeout:     10, //srv-prx-cli.Timeout
		MaxIncomingStreams: 10000000,
		KeepAlivePeriod:    10 * time.Second,
	}

	go func() {
		err := prx.InitProxy("../cfg/poa/routing/cfg_proxy.json", quicConf, routing.InitFixed)
		if err != nil {
			return
		}
	}()

	// start server in a separate goroutine
	go func() {
		if err := srv.InitServer(quicConf, server.InitReply); err != nil {
			log.Fatalf("Error running server: %v", err)
		}
	}()

	time.Sleep(5 * time.Second)

	// Inizializza un'istanza del client
	cli := &client.Client{}

	// Specifica il percorso del file di configurazione
	configFile := "../cfg/poa/client/cfg.json"

	// InitClient
	err := cli.InitClient(configFile, quicConf, client.InitFixed)
	if err != nil {
		log.Fatalf("Errore durante l'inizializzazione del client: %v", err)
	}
}
