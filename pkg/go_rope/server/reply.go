package server

import (
	"fmt"
	"go_rope/util"
	"time"

	"github.com/pelletier/go-toml"
	"github.com/quic-go/quic-go"
)

// Sinks
var d []string

// rtt
var rtt []float64

// Average service time
var service time.Duration

// Number of packets sent back
var pkts int64

// Counter
var c, r int64

// For computing moving average
var stime = time.Time{}
var obswindow = 10 * time.Second

const tw = 60

var bUp = util.NewMavg(tw)
var bDown = util.NewMavg(tw)

// func InitReply(workTime interface{}, dests interface{}, rtts interface{}, packets interface{}) {
func InitReply(conf *toml.Tree) {

	dests := conf.Get("variables.destination")
	// rtt := config.Get("variables.rtt")
	workTime := conf.Get("variables.workTime")
	packets := conf.Get("variables.packets")

	// InitReply(wt, dest, rtt, pkts)
	ForwardDecision = ReplyDecision
	ForwardBlock = nil
	ForwardSetLastResponse = ReplySetLastResponse
	// Initialize app configuration
	service, _ = time.ParseDuration(workTime.(string))
	pkts = packets.(int64)
	if dests != nil {
		// rs := rtts.([]interface{})
		ds := dests.([]interface{})
		fmt.Println("Using destinations:")
		for i, di := range ds {
			// fmt.Printf("%d - %s:\t %f", i, di, rs[i])
			fmt.Printf("%d - %s", i, di)
			// rtt = append(rtt, rs[i].(float64))
			d = append(d, ds[i].(string))
		}
	} else {
		fmt.Println("No destinations provided!!")
	}

	// Initialize counters
	c = 0
	r = 0
}

func ReplyDecision(req *util.RoPEMessage, session *map[string]quic.EarlyConnection, i int64) bool {
	// Emulate processing time
	time.Sleep(service)
	req.Body = make([]byte, req.ResSize)
	req.Type = util.Response
	tmp := req.Source
	req.Source = req.Destination
	req.Destination = tmp
	c += int64(len(req.Body))
	if (stime == time.Time{}) {
		stime = time.Now()
	} else {
		if time.Now().After(stime.Add(obswindow)) {
			stime = time.Now()
			util.Mavg_push(&bUp, c)
			util.Mavg_push(&bDown, r)

			c = 0
			r = 0

			fmt.Printf("Uplink:  %f \n", util.Mavg_eval(bUp, int64(obswindow/time.Second)))
			fmt.Printf("Downlink:  %f \n", util.Mavg_eval(bDown, int64(obswindow/time.Second)))
		}
	}

	return false
}

func ReplySetLastResponse(lastResp *util.RoPEMessage) {
	if lastResp.Type == util.Response {
		r += int64(lastResp.ResSize)
	}
}
