package main

import (
	"fmt"
	"log"
	"time"

	"github.com/paolocastagno/go_rope/pkg/client"
	"github.com/paolocastagno/go_rope/pkg/config"
	"github.com/paolocastagno/go_rope/pkg/util"

	"github.com/pelletier/go-toml"
	"github.com/quic-go/quic-go"
)

type app_cfg struct {
	destAddr string
	size     int64
	res_size int64
	rcv      int64
	snt      int64
	rcv_w    int64
	snt_w    int64
}

var (
	cli_app app_cfg
	stime   = time.Time{} // For computing moving average

	obswindow       = 10 * time.Second
	tw        int64 = 60

	bUp   = util.NewMavg(tw)
	bDown = util.NewMavg(tw)
)

func InitFixed(conf *toml.Tree) {

	// Read the destination address from app configuration
	dest := conf.Get("variables.destination")
	reqsize := conf.Get("variables.requestSize")
	ressize := conf.Get("variables.responseSize")

	if dest == nil {
		config.Die("No destination address specified")
	} else {
		cli_app.destAddr = dest.(string)
	}
	cli_app.size = reqsize.(int64)
	cli_app.res_size = ressize.(int64)

	fmt.Println("Loading logic fixed")
	fmt.Printf("\t- destination %s\n\t- request size %d bytes\n\t- response size %d bytes\n", dest, cli_app.size, cli_app.res_size)

	client.ForwardDecision = func(msg *util.RoPEMessage, destinations []string) string {
		return FixedDecision(msg, dest.(string))
	}
	client.ForwardSetLastResponse = FixedSetLastResponse
}

func FixedDecision(msg *util.RoPEMessage, dest string) string { ///
	cli_app.snt_w += cli_app.res_size
	msg.ResSize = int32(cli_app.res_size)
	msg.Body = make([]byte, cli_app.size)
	msg.Destination = dest
	fmt.Println("dest ", msg.Destination)
	//msg.Destination = destinations[0] //

	if (stime == time.Time{}) {
		stime = time.Now()
	} else {
		if time.Now().After(stime.Add(obswindow)) {
			stime = time.Now()

			util.Mavg_push(&bUp, cli_app.snt_w)
			util.Mavg_push(&bDown, cli_app.rcv_w)

			cli_app.rcv += cli_app.rcv_w
			cli_app.snt += cli_app.snt_w

			cli_app.snt_w = 0
			cli_app.rcv_w = 0

			fmt.Printf("Uplink:  %f \n", util.Mavg_eval(bUp, int64(obswindow/time.Second)))
			fmt.Printf("Downlink:  %f \n", util.Mavg_eval(bDown, int64(obswindow/time.Second)))
		}
	}
	return cli_app.destAddr
}

func FixedSetLastResponse(lastResp util.RoPEMessage) {
	if lastResp.Type == util.Response {
		cli_app.rcv += int64(lastResp.ResSize)
	}
}

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
