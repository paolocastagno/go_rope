package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/paolocastagno/go_rope/pkg/client"
	"github.com/paolocastagno/go_rope/pkg/util"
	"github.com/pelletier/go-toml"
	"github.com/quic-go/quic-go"

	"encoding/binary"
)

type appCfg struct {
	destAddr []string
	size     int64
	resSize  int64
	rcv      int64
	snt      int64
	rcvW     int64
	sntW     int64
}

var (
	stime = time.Time{} // For computing moving average

	obsWindow        = 10 * time.Second
	timeWindow int64 = 60

	bUp       = util.NewMavg(timeWindow)
	bDown     = util.NewMavg(timeWindow)
	histogram *util.Histogram
)

func InitFixed(conf *toml.Tree) *interface{} {
	// Create a local appCfg instance
	app := new(appCfg)

	// Safely retrieve the destinations from the application section
	destinations, ok := conf.Get("destinations").([]interface{})
	if !ok || len(destinations) == 0 {
		log.Fatalf("No destination addresses specified or invalid format")
	}

	// Convert []interface{} to []string
	app.destAddr = make([]string, len(destinations))
	for i, d := range destinations {
		str, ok := d.(string)
		if !ok {
			log.Fatalf("Invalid destination value at index %d", i)
		}
		app.destAddr[i] = str
	}

	// Retrieve request and response sizes
	reqSize, ok := conf.Get("requestSize").(int64)
	if !ok {
		log.Fatalf("Invalid or missing 'requestSize'")
	}
	resSize, ok := conf.Get("responseSize").(int64)
	if !ok {
		log.Fatalf("Invalid or missing 'responseSize'")
	}

	app.size = reqSize
	app.resSize = resSize

	fmt.Println("Loading logic fixed")
	fmt.Printf("\t- destinations %v\n\t- request size %d bytes\n\t- response size %d bytes\n", app.destAddr, app.size, app.resSize)

	client.ForwardDecision = func(msg *util.RoPEMessage, destinations []string, application *interface{}) string {
		// Use the first destination for simplicity
		return FixedDecision(msg, app.destAddr[0], application)
	}
	client.ForwardSetLastResponse = func(lastResp util.RoPEMessage, application *interface{}) {
		FixedSetLastResponse(lastResp, application)
	}

	// Return the appCfg instance as a pointer to an interface
	appInterface := new(interface{})
	*appInterface = app
	return appInterface
}

func FixedDecision(msg *util.RoPEMessage, dest string, cfg *interface{}) string {
	app, err := toAppCfg(cfg)
	if err != nil {
		log.Fatalf("FixedDecision: Error converting interface to appCfg: %v", err)
	}

	app.sntW += app.resSize
	msg.ResSize = int32(app.resSize)

	// Initialize the Body field with a length of at least 8 bytes
	msg.Body = make([]byte, app.size)

	// Serialize the timestamp into the Body field
	timestamp := time.Now().UnixNano()
	binary.BigEndian.PutUint64(msg.Body[:8], uint64(timestamp))

	msg.Destination = dest

	if stime == (time.Time{}) {
		stime = time.Now()
	} else if time.Now().After(stime.Add(obsWindow)) {
		stime = time.Now()

		util.Mavg_push(&bUp, app.sntW)
		util.Mavg_push(&bDown, app.rcvW)

		app.rcv += app.rcvW
		app.snt += app.sntW

		app.sntW = 0
		app.rcvW = 0

		fmt.Printf("Uplink:  %f \n", util.Mavg_eval(bUp, int64(obsWindow/time.Second)))
		fmt.Printf("Downlink:  %f \n", util.Mavg_eval(bDown, int64(obsWindow/time.Second)))
	}
	return dest
}

func FixedSetLastResponse(lastResp util.RoPEMessage, cfg *interface{}) {
	app, err := toAppCfg(cfg)
	if err != nil {
		log.Fatalf("FixedSetLastResponse: Error converting interface to appCfg: %v", err)
	}

	if lastResp.Type == util.Response {
		app.rcv += int64(lastResp.ResSize)

		// Extract the timestamp from the Body field
		if len(lastResp.Body) < 8 {
			log.Printf("Invalid Body length: %d", len(lastResp.Body))
			return
		}
		ts := binary.BigEndian.Uint64(lastResp.Body[:8])
		timestamp := time.Unix(0, int64(ts))

		// Compute RTT
		fmt.Println(timestamp.Format(time.RFC850))
		fmt.Println(time.Now().Format(time.RFC850))
		rtt := time.Since(timestamp)
		fmt.Printf("RTT: %v\n", rtt)

		// Add RTT to the histogram
		histogram.Add(rtt.Seconds())
	}
}

func exponentialTicker(rps float64) *time.Ticker {
	interval := time.Duration(float64(time.Second) / rps)
	fmt.Printf("Generating requests at an interval of %v (%.2f requests per second)\n", interval, rps)
	return time.NewTicker(interval)
}

// func exponentialTicker(rps float64) *jitterbug.Ticker {
// 	fmt.Println("Tuning value to get", rps, "requests per second.")
// 	rand.Seed(uint64(time.Now().UTC().UnixNano())) // Cast int64 to uint64

// 	beta := 1.0 / rps
// 	return jitterbug.New(
// 		time.Millisecond*0,
// 		&jitterbug.Univariate{
// 			Sampler: &distuv.Gamma{
// 				Alpha: 1,
// 				Beta:  beta,
// 			},
// 		},
// 	)
// }

func toAppCfg(data *interface{}) (*appCfg, error) {
	cfg := new(appCfg)
	ok := false
	cfg, ok = (*data).(*appCfg)
	if !ok {
		return &appCfg{}, fmt.Errorf("failed to convert interface to appCfg")
	}
	return cfg, nil
}

func main() {
	// Initialize the histogram with a bin size of 0.01 seconds (10 ms)
	histogram = util.NewHistogram(0.001)

	// Parse command-line arguments
	configFile := flag.String("config", "", "Path to the configuration file")
	flag.Parse()
	if *configFile == "" {
		log.Fatal("Configuration file is required")
	} else {
		fmt.Printf("Configuration file: %s\n", *configFile)
	}
	cli := &client.Client{}

	quicConf := &quic.Config{
		MaxIdleTimeout:     10 * time.Second,
		MaxIncomingStreams: 10000000,
		KeepAlivePeriod:    10 * time.Second,
	}

	// Initialize the client
	if err := cli.InitClient(quicConf, *configFile, InitFixed); err != nil {
		log.Fatalf("Error initializing client: %v", err)
	}

	// Set up ticker and channels
	ticker := exponentialTicker(1) // Example: 1 request per second
	counter := make(chan int64, cli.MaxConcurrentConnections)

	// Set up graceful shutdown
	util.SetupGracefulShutdown(func() {
		cli.Done <- true
	})

	var wg sync.WaitGroup

	// Main loop
	for {
		select {
		case <-cli.Done:
			// Stop the ticker and wait for all goroutines to finish
			ticker.Stop()
			wg.Wait()
			histogram.Print() // Print the histogram at the end
			fmt.Println("Client execution stopped.")
			return
		case <-ticker.C:
			// Limit the number of concurrent requests
			if len(counter) == cap(counter) {
				fmt.Printf("maxConcurrentConnections=%d reached\n", cli.MaxConcurrentConnections)
				continue
			}

			// Generate a new request
			id := time.Now().UnixNano()
			counter <- id
			wg.Add(1)
			go func(id int64) {
				defer wg.Done()
				defer func() { <-counter }()
				if err := cli.NewReq(cli.Connections, id); err != nil {
					log.Printf("Error in request: %v", err)
				}
			}(id)
		}
	}
}
