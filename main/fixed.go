package main

import (
	"fmt"
	"time"

	"github.com/pelletier/go-toml"
)

// Counter
var cu, cd int64

// For computing moving average
var stim = time.Time{}
var obswind = 10 * time.Second

const twind = 100

var b_u = util.NewMavg(twind)
var b_d = util.NewMavg(twind)

var rtr *routing.Proxy

func InitFixed(conf *toml.Tree, rtr_p *routing.Proxy) {
	rtr = rtr_p
	dest := conf.Get("variables.dest")

	if dest == nil {
		config.Die("No destination specified")
	}
	dhost := dest.(string)
	fmt.Printf("Fixed routing toward:\t %s\n", dhost)
	// Initialize counters
	cu = 0
	cd = 0

	rtr.ForwardDecision = FixedDecision
	rtr.ForwardSetLastResponse = FixedSetLastResponse
}

func FixedDecision(req *util.RoPEMessage) {
	cu += int64(len(req.Body))

	if (stim == time.Time{}) {
		stim = time.Now()
	} else {
		if time.Now().After(stim.Add(obswind)) {
			stim = time.Now()
			util.Mavg_push(&b_u, cu)
			util.Mavg_push(&b_d, cd)

			fmt.Printf("\nUplink:  %f \n", util.Mavg_eval(b_u, int64(obswind/time.Second)))
			for i, s := range rtr.d {
				fmt.Printf("\tUplink %s:  %f bytes/s\n", s, util.Mavg_eval(rtr.b_up_i[i], int64(obswind/time.Second)))
			}
			fmt.Printf("Downlink:  %f \n", util.Mavg_eval(b_d, int64(obswind/time.Second)))
		}
	}
}

func FixedSetLastResponse(lastResp *util.RoPEMessage) {
	if lastResp.Type == util.Response {
		cd += int64(len(lastResp.Body))
	}
}
