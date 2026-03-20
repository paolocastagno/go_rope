package routing

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/paolocastagno/go_rope/pkg/util"
)

// Routing probabilities
var routingProb []float64

// Sinks
var sinks []string

// Counter
var countup, countdown []int64

// Latency
var distribution_up map[string]interface{}
var distributionType_up map[string]string
var distribution_down map[string]interface{}
var distributionType_down map[string]string

// For computing moving average
var st = time.Time{}
var obsw = 1 * time.Second

const twindow = 60

var byte_up util.Mavg = util.NewMavg(twindow)
var byte_down util.Mavg = util.NewMavg(twindow)
var byte_up_i []util.Mavg
var byte_down_i []util.Mavg

// Mutex for thread-safe access to all latency routing state
var plMutex sync.RWMutex

func InitPL(probs interface{}, dest interface{}, latencyParams_up interface{}, latencyType_up interface{}, latencyParams_down interface{}, latencyType_down interface{}) {

	if probs == nil || latencyType_up == nil || latencyType_down == nil {
		die("No probability or latency type specified")
	}
	ps := probs.([]interface{})
	ds := dest.([]interface{})
	ltu := latencyType_up.([]interface{})
	lpu := latencyParams_up.([]interface{})
	ltd := latencyType_down.([]interface{})
	lpd := latencyParams_down.([]interface{})

	distribution_up = make(map[string]interface{})
	distributionType_up = make(map[string]string)

	distribution_down = make(map[string]interface{})
	distributionType_down = make(map[string]string)

	fmt.Printf("len(ps): %d\n", len(ps))
	fmt.Println("Using probabilities:")
	for i, pi := range ps {
		fmt.Printf("%s:\t %f\n", ds[i].(string), pi)
		// Initialize probabilities and destinations
		routingProb = append(routingProb, ps[i].(float64))
		sinks = append(sinks, ds[i].(string))
		// Initialize latency
		distUp, err := util.InitDelay(ltu[i].(string), lpu[i])
		if err != nil {
			fmt.Printf("Error initializing up latency for %s: %v\n", sinks[i], err)
		}
		distribution_up[sinks[i]] = distUp
		distributionType_up[sinks[i]] = ltu[i].(string)

		distDown, err := util.InitDelay(ltd[i].(string), lpd[i])
		if err != nil {
			fmt.Printf("Error initializing down latency for %s: %v\n", sinks[i], err)
		}
		distribution_down[sinks[i]] = distDown
		distributionType_down[sinks[i]] = ltd[i].(string)
		// Initialize counters
		countup = append(countup, 0)
		countdown = append(countdown, 0)
		// Initialize moving averages
		byte_up_i = append(byte_up_i, util.NewMavg(twindow))
		byte_down_i = append(byte_down_i, util.NewMavg(twindow))
	}
}

func PLDecision(req *util.RoPEMessage) {
	plMutex.Lock()
	defer plMutex.Unlock()

	if len(routingProb) == 0 {
		fmt.Println("No routing probabilities configured")
		return
	}

	res := rand.Float64()
	var i = 0
	var pdest = routingProb[0]
	for i < (len(routingProb)-1) && pdest < res {
		i++
		pdest += routingProb[i]
	}
	req.Hop = req.Destination
	req.Destination = sinks[i]

	// Delay the incoming request
	if err := util.Delay(distribution_up[req.Destination], distributionType_up[req.Destination]); err != nil {
		fmt.Printf("Error applying delay: %v\n", err)
	}

	countup[i] += int64(len(req.Body))

	if (st == time.Time{}) {
		st = time.Now()
	} else {

		if time.Now().After(st.Add(obsw)) {
			st = time.Now()
			var totalup, totaldown int64 = 0, 0
			for i := range byte_up_i {
				totalup += countup[i]
				util.Mavg_push(&byte_up_i[i], countup[i])
				countup[i] = 0
				totaldown += countdown[i]
				util.Mavg_push(&byte_down_i[i], countdown[i])
				countdown[i] = 0
			}
			util.Mavg_push(&byte_up, totalup)
			util.Mavg_push(&byte_down, totaldown)

			fmt.Printf("Uplink:  %f \n", util.Mavg_eval(byte_up, int64(obsw/time.Second)))
			for i, s := range sinks {
				fmt.Printf("\tUplink %s:  %f bytes/s\n", s, util.Mavg_eval(byte_up_i[i], int64(obsw/time.Second)))
			}
			fmt.Printf("Downlink:  %f \n", util.Mavg_eval(byte_down, int64(obsw/time.Second)))
			for i, s := range sinks {
				fmt.Printf("\tDownlink %s:  %f bytes/s\n", s, util.Mavg_eval(byte_down_i[i], int64(obsw/time.Second)))
			}
		}
	}
}

func PLSetLastResponse(lastResp *util.RoPEMessage) {
	plMutex.Lock()
	defer plMutex.Unlock()

	if lastResp.Type == util.Response {
		if err := util.Delay(distribution_down[lastResp.Source], distributionType_down[lastResp.Source]); err != nil {
			fmt.Printf("Error applying down delay: %v\n", err)
		}
		var i = 0
		for i < len(sinks) && sinks[i] != lastResp.Destination {
			i++
		}
		if i < len(countdown) {
			countdown[i] += int64(len(lastResp.Body))
		}
	}
}
