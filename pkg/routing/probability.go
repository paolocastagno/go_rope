package routing

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/paolocastagno/go_rope/pkg/util"
)

func die(msg ...interface{}) {
	fmt.Println(msg...)
	os.Exit(1)
}

// ProbabilityRouter holds all shared state with mutex protection
type ProbabilityRouter struct {
	mu        sync.RWMutex
	p         []float64       // Routing probabilities
	d         []string        // Sinks/destinations
	cup, cdw  []int64         // Counters
	stime     time.Time       // Start time for observation window
	obswindow time.Duration   // Observation window
	b_up      util.Mavg       // Overall uplink moving average
	b_down    util.Mavg       // Overall downlink moving average
	b_up_i    []util.Mavg     // Per-destination uplink
	b_down_i  []util.Mavg     // Per-destination downlink
}

const timewindow = 60

var probRouter = &ProbabilityRouter{
	obswindow: 10 * time.Second,
	b_up:      util.NewMavg(timewindow),
	b_down:    util.NewMavg(timewindow),
}

// Keep globals for backward compatibility (deprecated)
var p []float64
var d []string
var cup, cdw []int64
var stime = time.Time{}
var obswindow = 10 * time.Second
var b_up = util.NewMavg(timewindow)
var b_down = util.NewMavg(timewindow)
var b_up_i []util.Mavg
var b_down_i []util.Mavg

func InitWeightedRandom(probs interface{}, dest interface{}) {
	probRouter.mu.Lock()
	defer probRouter.mu.Unlock()

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
		probRouter.p = append(probRouter.p, ps[i].(float64))
		probRouter.d = append(probRouter.d, ds[i].(string))
		// Initialize counters
		probRouter.cup = append(probRouter.cup, 0)
		probRouter.cdw = append(probRouter.cdw, 0)
		// Initialize moving averages
		probRouter.b_up_i = append(probRouter.b_up_i, util.NewMavg(timewindow))
		probRouter.b_down_i = append(probRouter.b_down_i, util.NewMavg(timewindow))
	}

	// Update globals for backward compatibility
	p = probRouter.p
	d = probRouter.d
	cup = probRouter.cup
	cdw = probRouter.cdw
	b_up_i = probRouter.b_up_i
	b_down_i = probRouter.b_down_i
}

func WeightedRandomDecision(req *util.RoPEMessage) {
	probRouter.mu.Lock()
	defer probRouter.mu.Unlock()

	if len(probRouter.p) == 0 {
		fmt.Println("No probabilities configured")
		return
	}

	res := rand.Float64()
	var i int = 0
	var pdest float64 = probRouter.p[0]
	for i < (len(probRouter.p)-1) && pdest < res {
		i++
		pdest += probRouter.p[i]
	}
	probRouter.cup[i] += int64(len(req.Body))
	if (probRouter.stime == time.Time{}) {
		probRouter.stime = time.Now()
	} else {
		if time.Now().After(probRouter.stime.Add(probRouter.obswindow)) {
			probRouter.stime = time.Now()
			var totalup, totaldown int64 = 0, 0
			for i := range probRouter.b_up_i {
				totalup += probRouter.cup[i]
				util.Mavg_push(&probRouter.b_up_i[i], probRouter.cup[i])
				probRouter.cup[i] = 0
				totaldown += probRouter.cdw[i]
				util.Mavg_push(&probRouter.b_down_i[i], probRouter.cdw[i])
				probRouter.cdw[i] = 0
			}
			util.Mavg_push(&probRouter.b_up, totalup)
			util.Mavg_push(&probRouter.b_down, totaldown)

			fmt.Printf("Uplink:  %f \n", util.Mavg_eval(probRouter.b_up, int64(probRouter.obswindow/time.Second)))
			for i, s := range probRouter.d {
				fmt.Printf("\tUplink %s:  %f bytes/s\n", s, util.Mavg_eval(probRouter.b_up_i[i], int64(probRouter.obswindow/time.Second)))
			}
			fmt.Printf("Downlink:  %f \n", util.Mavg_eval(probRouter.b_down, int64(probRouter.obswindow/time.Second)))
			for i, s := range probRouter.d {
				fmt.Printf("\tDownlink %s:  %f bytes/s\n", s, util.Mavg_eval(probRouter.b_down_i[i], int64(probRouter.obswindow/time.Second)))
			}
		}
	}
	req.Hop = req.Destination
	req.Destination = probRouter.d[i]
}

func WeightedRandomSetLastResponse(lastResp *util.RoPEMessage) {
	probRouter.mu.Lock()
	defer probRouter.mu.Unlock()

	if lastResp.Type == util.Response {
		var i int = 0
		for i < len(probRouter.d) && probRouter.d[i] != lastResp.Source {
			i++
		}
		if i < len(probRouter.cdw) {
			probRouter.cdw[i] += int64(len(lastResp.Body))
		}
	}
}
