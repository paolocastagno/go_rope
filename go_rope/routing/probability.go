package routing

import (
	"fmt"
	"math/rand"
	"time"

	//modulo locale
	"github.com/paolocastagno/go_rope/util"
)

// Routing probabilities
var p []float64

// Sinks
var d []string

// Counter
var cup, cdw []int64

// For computing moving average
var stime = time.Time{}
var obswindow = 10 * time.Second

const timewindow = 60

var b_up = util.NewMavg(timewindow)
var b_down = util.NewMavg(timewindow)
var b_up_i []util.Mavg
var b_down_i []util.Mavg

func InitWeightedRandom(probs interface{}, dest interface{}) {

	if probs == nil {
		die("No probability specified")
	}
	ps := probs.([]interface{})
	ds := dest.([]interface{})
	fmt.Printf("len(ps): %d\n", len(ps))
	fmt.Println("Using probabilities:")
	for i, pi := range ps {
		fmt.Printf("%d:\t %f", i, pi)
		// Initialize probabilities and destinations
		p = append(p, ps[i].(float64))
		d = append(d, ds[i].(string))
		// Initialize counters
		cup = append(cup, 0)
		cdw = append(cdw, 0)
		// Initialize moving averages
		b_up_i = append(b_up_i, util.NewMavg(timewindow))
		b_down_i = append(b_down_i, util.NewMavg(timewindow))
	}
}

func WeightedRandomDecision(req *util.RoPEMessage) {
	res := rand.Float64()
	var i int = 0
	var pdest float64 = p[0]
	for i < (len(p)-1) && pdest < res {
		i++
		pdest += p[i]
	}
	cup[i] += int64(len(req.Body))
	if (stime == time.Time{}) {
		stime = time.Now()
	} else {
		if time.Now().After(stime.Add(obswindow)) {
			stime = time.Now()
			var totalup, totaldown int64 = 0, 0
			for i := range b_up_i {
				totalup += cup[i]
				util.Mavg_push(&b_up_i[i], cup[i])
				cup[i] = 0
				totaldown += cdw[i]
				util.Mavg_push(&b_down_i[i], cdw[i])
				cdw[i] = 0
			}
			util.Mavg_push(&b_up, totalup)
			util.Mavg_push(&b_down, totaldown)

			fmt.Printf("Uplink:  %f \n", util.Mavg_eval(b_up, int64(obswindow/time.Second)))
			for i, s := range d {
				fmt.Printf("\tUplink %s:  %f bytes/s\n", s, util.Mavg_eval(b_up_i[i], int64(obswindow/time.Second)))
			}
			fmt.Printf("Downlink:  %f \n", util.Mavg_eval(b_down, int64(obswindow/time.Second)))
			for i, s := range d {
				fmt.Printf("\tDownlink %s:  %f bytes/s\n", s, util.Mavg_eval(b_down_i[i], int64(obswindow/time.Second)))
			}
		}
	}
	req.Hop = req.Destination
	req.Destination = d[i]
}

func WeightedRandomSetLastResponse(lastResp *util.RoPEMessage) {
	if lastResp.Type == util.Response {
		var i int = 0
		for i < len(d) && d[i] != lastResp.Source {
			i++
		}
		cdw[i] += int64(len(lastResp.Body))
	}
}
