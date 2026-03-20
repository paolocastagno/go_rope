package routing

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/paolocastagno/go_rope/pkg/util"

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

// Mutex for thread-safe access
var fixedMutex sync.RWMutex

func Die(msg ...interface{}) {
	fmt.Println(msg...)
	os.Exit(1)
}

func InitFixed(conf *toml.Tree, proxy *Proxy) {

	dest := conf.Get("variables.dest")

	if dest == nil {
		Die("No destination specified")
	}
	dhost := dest.(string)
	fmt.Printf("Fixed routing toward:\t %s\n", dhost)
	// Initialize counters
	cu = 0
	cd = 0

	proxy.ForwardDecision = FixedDecision
	proxy.ForwardSetLastResponse = FixedSetLastResponse
}

func FixedDecision(req *util.RoPEMessage) {
	fixedMutex.Lock()
	defer fixedMutex.Unlock()

	cu += int64(len(req.Body))

	if (stim == time.Time{}) {
		stim = time.Now()
	} else {
		if time.Now().After(stim.Add(obswind)) {
			stim = time.Now()
			util.Mavg_push(&b_u, cu)
			util.Mavg_push(&b_d, cd)

			fmt.Printf("\nUplink:  %f \n", util.Mavg_eval(b_u, int64(obswind/time.Second)))
			fmt.Printf("Downlink:  %f \n", util.Mavg_eval(b_d, int64(obswind/time.Second)))
		}
	}
}

func FixedSetLastResponse(lastResp *util.RoPEMessage) {
	fixedMutex.Lock()
	defer fixedMutex.Unlock()

	if lastResp.Type == util.Response {
		cd += int64(len(lastResp.Body))
	}
}
