package client

import (
	"fmt"
	"go_rope/config"
	"go_rope/util"
	"time"

	"github.com/pelletier/go-toml"
)

var destAddr = ""
var size, rsize int32 = 0, 0
var sent, received int64 = 0, 0 // Counters
var stime = time.Time{}         // For computing moving average
var obswindow = 10 * time.Second

const tw = 60

var bUp = util.NewMavg(tw)
var bDown = util.NewMavg(tw)

func InitFixed(conf *toml.Tree) {

	// Read the destination address from app configuration
	dest := conf.Get("variables.destination")
	reqsize := conf.Get("variables.requestSize")
	ressize := conf.Get("variables.responseSize")

	if dest == nil {
		config.Die("No destination address specified")
	} else {
		destAddr = dest.(string)
	}
	size = int32(reqsize.(int64))
	rsize = int32(ressize.(int64))
	fmt.Println("Loading logic fixed")
	fmt.Printf("\t- destination %s\n\t- request size %d bytes\n\t- response size %d bytes\n", dest, size, rsize)

	ForwardDecision = func(msg *util.RoPEMessage, destinations []string) string {
		return FixedDecision(msg, dest.(string))
	}
	ForwardSetLastResponse = FixedSetLastResponse
}

func FixedDecision(msg *util.RoPEMessage, dest string) string { ///
	sent += int64(rsize)
	msg.ResSize = rsize
	msg.Body = make([]byte, size)
	msg.Destination = dest
	fmt.Println("dest ", msg.Destination)
	//msg.Destination = destinations[0] //

	if (stime == time.Time{}) {
		stime = time.Now()
	} else {
		if time.Now().After(stime.Add(obswindow)) {
			stime = time.Now()

			util.Mavg_push(&bUp, sent)
			util.Mavg_push(&bDown, received)

			sent = 0
			received = 0

			fmt.Printf("Uplink:  %f \n", util.Mavg_eval(bUp, int64(obswindow/time.Second)))
			fmt.Printf("Downlink:  %f \n", util.Mavg_eval(bDown, int64(obswindow/time.Second)))
		}
	}
	return destAddr
}

func FixedSetLastResponse(lastResp util.RoPEMessage) {
	if lastResp.Type == util.Response {
		received += int64(lastResp.ResSize)
	}
}
