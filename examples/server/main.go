package main

import (
	"fmt"
	"log"
	"time"

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
	btime   time.Time

	// For computing moving average
	stime     = time.Time{}
	obswindow = 10 * time.Second

	tw int64 = 60

	bUp   = util.NewMavg(tw)
	bDown = util.NewMavg(tw)
)

func InitReply(conf *toml.Tree) {
	fmt.Print(conf)
	srv_app.procTime, _ = time.ParseDuration(conf.Get("variables.processing_time").(string))
	srv_app.packets = conf.Get("variables.packets").(int64)
	srv_app.packetSize = conf.Get("variables.packet_size").(int64)

	server.ForwardDecision = ReplyDecision
	server.ForwardBlock = nil
	server.ForwardSetLastResponse = ReplySetLastResponse
}

func ReplyDecision(req *util.RoPEMessage, session *map[string]quic.EarlyConnection, i int64) bool {
	// Emulate processing time
	time.Sleep(srv_app.procTime)
	req.Body = make([]byte, req.ResSize)
	req.Type = util.Response
	tmp := req.Source
	req.Source = req.Destination
	req.Destination = tmp
	srv_app.rcv += int64(len(req.Body))
	if (stime == time.Time{}) {
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
	srv := server.NewServer(
		"Server",
		"localhost:8080",
		"app_server.toml",
		10,
		5,
		30*time.Second,
	)
	server.ForwardDecision = ReplyDecision
	server.ForwardSetLastResponse = ReplySetLastResponse

	quicConf := &quic.Config{
		MaxIdleTimeout:     10 * time.Second,
		MaxIncomingStreams: 10000000,
		KeepAlivePeriod:    10 * time.Second,
	}

	if err := srv.InitServer(quicConf, InitReply); err != nil {
		log.Fatalf("Error running server: %v", err)
	}
}
