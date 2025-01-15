package main

import (
    "log"
    "time"

    "github.com/paolocastagno/go_rope/pkg/client"
    "github.com/quic-go/quic-go"
)

func main() {
    cli := &client.Client{}

    configFile := "cfg.json"
    quicConf := &quic.Config{
        MaxIdleTimeout:     10 * time.Second,
        MaxIncomingStreams: 10000000,
        KeepAlivePeriod:    10 * time.Second,
    }

    if err := cli.InitClient(configFile, quicConf, InitFixed); err != nil {
        log.Fatalf("Error initializing client: %v", err)
    }
}