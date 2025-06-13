package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"encoding/binary"

	"github.com/paolocastagno/go_rope/pkg/server"
	"github.com/paolocastagno/go_rope/pkg/util"
	"github.com/pelletier/go-toml"
	"github.com/quic-go/quic-go"
)

type app_cfg struct {
	procTime   time.Duration
	packets    int64
	packetSize int64
	rcv        int64
	snt        int64
	rcv_w      int64
	snt_w      int64
}

var (
	srv_app app_cfg

	// For computing moving average
	stime     = time.Time{}
	obswindow = 10 * time.Second

	tw int64 = 60

	bUp   = util.NewMavg(tw)
	bDown = util.NewMavg(tw)
)

func InitReply(conf *toml.Tree) *interface{} {
	fmt.Print(conf)

	// Initialize srv_app
	srv_app.packets = 1 // Default value
	if packets, ok := conf.Get("Packets").(int64); ok {
		srv_app.packets = packets
	} else {
		fmt.Println("Warning: 'application.Packets' is missing or invalid, using default value 1")
	}

	srv_app.packetSize = 64 // Default value
	if packetSize, ok := conf.Get("Packet_size").(int64); ok {
		srv_app.packetSize = packetSize
	} else {
		fmt.Println("Warning: 'application.Packet_size' is missing or invalid, using default value 64")
	}

	// Return the application configuration
	srvinterface := new(interface{})
	*srvinterface = srv_app
	return srvinterface
}

func ReplyDecision(req *util.RoPEMessage, session *map[string]quic.EarlyConnection, i int64) bool {
	// Emulate processing time
	time.Sleep(srv_app.procTime)

	// Extract the timestamp from the request's Body field
	var timestamp uint64
	if len(req.Body) >= 8 {
		timestamp = binary.BigEndian.Uint64(req.Body[:8]) // Deserialize the timestamp
	} else {
		fmt.Println("Warning: Request Body is too short to contain a timestamp")
		timestamp = 0 // Default to zeroed timestamp
	}

	// Prepare the response
	req.Body = make([]byte, req.ResSize)
	binary.BigEndian.PutUint64(req.Body[:8], timestamp) // Serialize the timestamp into the response's Body field

	// If the response is fragmented, ensure the timestamp is copied into all fragments
	for i := 0; i < len(req.Body); i += int(srv_app.packetSize) {
		end := i + int(srv_app.packetSize)
		if end > len(req.Body) {
			end = len(req.Body)
		}
		copy(req.Body[i:i+8], req.Body[:8]) // Copy the timestamp into each fragment
	}

	req.Type = util.Response

	// Swap Source and Destination
	tmp := req.Source
	req.Source = req.Destination
	req.Destination = tmp

	// Update server statistics
	srv_app.rcv += int64(len(req.Body))
	if stime == (time.Time{}) {
		stime = time.Now()
	} else {
		if time.Now().After(stime.Add(obswindow)) {
			stime = time.Now()
			util.Mavg_push(&bUp, srv_app.snt_w)
			util.Mavg_push(&bDown, srv_app.rcv_w)

			srv_app.rcv += srv_app.rcv_w
			srv_app.snt += srv_app.snt_w
			srv_app.snt_w = 0
			srv_app.rcv_w = 0

			fmt.Printf("Uplink:  %f \n", util.Mavg_eval(bUp, int64(obswindow/time.Second)))
			fmt.Printf("Downlink:  %f \n", util.Mavg_eval(bDown, int64(obswindow/time.Second)))
		}
	}

	return false
}

func ReplySetLastResponse(lastResp *util.RoPEMessage) {
	if lastResp.Type == util.Response {
		srv_app.snt_w += int64(lastResp.ResSize)
	}
}

func main() {
	// Parse command-line arguments
	configFile := flag.String("config", "", "Path to the configuration file")
	flag.Parse()
	if *configFile == "" {
		log.Fatal("Configuration file is required")
	} else {
		fmt.Printf("Configuration file: %s\n", *configFile)
	}
	// Create a new server instance
	srv := new(server.Server)

	// QUIC configuration
	quicConf := &quic.Config{
		MaxIdleTimeout:     10 * time.Second,
		MaxIncomingStreams: 10000000,
		KeepAlivePeriod:    10 * time.Second,
	}

	// Initialize the server
	err := server.InitServer(srv, quicConf, InitReply, *configFile)
	if err != nil {
		log.Fatalf("Error initializing server: %v", err)
	}

	// Run the server
	if err := server.Run(srv, quicConf, ReplyDecision, ReplySetLastResponse, nil); err != nil {
		log.Fatalf("Error running server: %v", err)
	}
}
