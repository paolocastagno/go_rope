package client

import (
	"context"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/lthibault/jitterbug"
	"github.com/pelletier/go-toml"
	"github.com/quic-go/quic-go"
	"gonum.org/v1/gonum/stat/distuv"

	"github.com/paolocastagno/go_rope/pkg/config"
	"github.com/paolocastagno/go_rope/pkg/util"
)

var GitCommit = "master"

// ForwardDecision is a function that decides the forwarding logic for a message
var ForwardDecision func(msg *util.RoPEMessage, destinations []string) string

// ForwardSetLastResponse is a function that sets the last response for a message
var ForwardSetLastResponse func(util.RoPEMessage)

// ForwardingLogic defines the interface for forwarding logic
type ForwardingLogic interface {
	Init()
	Decision(*util.RoPEMessage, []string) string
	SetLastResponse(util.RoPEMessage)
}

// Client represents the client configuration
type Client struct {
	IdDevice                 string
	Proxy                    string
	Destinations             []string
	RequestsPerSec           float64
	MaxConcurrentConnections uint
	TestDuration             time.Duration
	Timeout                  time.Duration
	Clicfg                   string
	Appcfg                   string
	LoggerEnabled            bool
	Sessions                 []quic.EarlyConnection
	Counter                  chan int64
	Wg                       sync.WaitGroup
}

// InitClient initializes the client with the given configuration file and QUIC configuration
func (client *Client) InitClient(configFile string, quicConf *quic.Config, initLogic func(*toml.Tree)) error {
	fmt.Printf("Running client version %s\n", GitCommit)

	// Load client parameters from the configuration file
	client.loadParam(configFile)
	fmt.Printf("Starting idDevice: %s\n", client.IdDevice)

	// Initialize the logger
	if err := util.InitLogger(); err != nil {
		panic(err)
	}
	defer util.CloseLogger()

	client.LoggerEnabled = util.IsLoggerEnabled()

	// Load forwarding policy if specified
	if client.Appcfg != "" {
		fmt.Println("Loading forwarding policy:", client.Appcfg)
		config.LoadForwardingConf(client.Appcfg, initLogic)
	} else {
		return errors.New("no forwarding policy specified")
	}

	// Start the client main process
	if err := client.clientMain(client.Destinations, quicConf); err != nil {
		fmt.Println("Error!", err)
	}

	fmt.Println("Finished! Waiting 20 seconds...")
	time.Sleep(20 * time.Second)

	return nil
}

// loadParam loads the client parameters from the given configuration file
func (client *Client) loadParam(config string) bool {
	jsonFile, err := os.Open(config)
	if err != nil {
		return false
	}
	defer jsonFile.Close()

	fmt.Println("Using config file:", config)
	byteValue, _ := io.ReadAll(jsonFile)

	var cfg map[string]interface{}
	if err := json.Unmarshal(byteValue, &cfg); err != nil {
		fmt.Println("error:", err)
		return false
	}

	client.IdDevice = getString(cfg, "IdDevice", "device_default")
	client.Proxy = getString(cfg, "Proxy", "")
	client.Destinations = append(client.Destinations, client.Proxy)
	client.RequestsPerSec = getFloat64(cfg, "RequestsPerSec", 1.0)
	client.MaxConcurrentConnections = uint(getFloat64(cfg, "MaxConcurrentConnections", 1.0))
	client.TestDuration = getDuration(cfg, "TestDuration", "0s")
	client.Timeout = getDuration(cfg, "Timeout", "0s")
	client.Appcfg = getString(cfg, "AppCfg", "")

	// Load logger configuration if specified
	if loggerCfg, ok := cfg["Logger"]; ok {
		byteValue, err := json.Marshal(loggerCfg)
		if err == nil {
			var logger util.LoggerConf
			if err := json.Unmarshal(byteValue, &logger); err == nil {
				util.SetLoggerParamFromConf(logger)
			}
		}
	}

	fmt.Printf("Client %s configuration:\n", client.IdDevice)
	client.printParams()
	return true
}

// getString retrieves a string value from the configuration map
func getString(cfg map[string]interface{}, key, defaultValue string) string {
	if value, ok := cfg[key].(string); ok {
		return value
	}
	return defaultValue
}

// getFloat64 retrieves a float64 value from the configuration map
func getFloat64(cfg map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := cfg[key].(float64); ok {
		return value
	}
	return defaultValue
}

// getDuration retrieves a duration value from the configuration map
func getDuration(cfg map[string]interface{}, key, defaultValue string) time.Duration {
	if value, ok := cfg[key].(string); ok {
		duration, err := time.ParseDuration(value)
		if err == nil {
			return duration
		}
	}
	duration, _ := time.ParseDuration(defaultValue)
	return duration
}

// printParams prints the client parameters
func (client *Client) printParams() {
	fmt.Println("Test duration:", client.TestDuration)
	fmt.Println("Timeout:", client.Timeout)
	fmt.Println("Destinations:", client.Destinations)
	fmt.Println("IdDevice:", client.IdDevice)
	fmt.Println("RequestsPerSec:", client.RequestsPerSec)
	fmt.Println("MaxConcurrentConnections:", client.MaxConcurrentConnections)
	fmt.Println("Appcfg:", client.Appcfg)
}

// setupTestDuration sets up the test duration timer
func (client *Client) setupTestDuration(done chan<- bool) {
	if client.TestDuration > 0 {
		fmt.Printf("Test duration set to %v\n", client.TestDuration)
		testTimer := time.NewTimer(client.TestDuration)
		go func() {
			<-testTimer.C
			fmt.Println("Test ended")
			done <- true
		}()
	} else {
		fmt.Println("No test duration set, running until stopped")
	}
}

// exponentialTicker creates a ticker that generates events at an exponentially distributed interval
func exponentialTicker(rps float64) *jitterbug.Ticker {
	fmt.Println("Tuning value to get", rps, "request per second.")
	rand.Seed(time.Now().UTC().UnixNano())

	beta := 0.000000001 * float64(rps)
	t := jitterbug.New(
		time.Millisecond*0,
		&jitterbug.Univariate{
			Sampler: &distuv.Gamma{
				Alpha: 1,
				Beta:  beta,
			},
		},
	)

	done := make(chan bool)
	testSec := uint(20)
	testTimer := time.NewTimer(time.Second * time.Duration(testSec))
	go func() {
		<-testTimer.C
		fmt.Println("Tuning ended")
		done <- true
	}()

	var counter float64
	start := time.Now()
	for {
		counter++
		select {
		case <-done:
			t.Stop()
			break
		case <-t.C:
		}
	}

	end := time.Now()
	fmt.Println("Duration:", end.Sub(start))
	fmt.Println("Expected", rps*float64(testSec), "requests in", testSec, "seconds")
	fmt.Println("Got", counter, "requests in", testSec, "seconds")

	newBeta := beta
	fmt.Println("New beta:", newBeta)

	return jitterbug.New(
		time.Millisecond*0,
		&jitterbug.Univariate{
			Sampler: &distuv.Gamma{
				Alpha: 1,
				Beta:  newBeta,
			},
		},
	)
}

// clientMain configures the QUIC connections with the destinations and handles requests
func (client *Client) clientMain(destinations []string, quicConf *quic.Config) error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"RoPEProtocol"},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "source", client.IdDevice)

	var wg sync.WaitGroup
	fmt.Printf("Requests per second set to %v\n", client.RequestsPerSec)
	fmt.Printf("Requests timeout set to %v\n", client.Timeout)

	var sessions []quic.EarlyConnection
	for _, d := range destinations {
		s, err := quic.DialAddrEarly(ctx, d, tlsConf, quicConf)
		if err != nil {
			fmt.Printf("Cannot connect to %s\n", d)
			log.Println(err)
			return err
		}
		sessions = append(sessions, s)
		fmt.Printf("Connected to %s\n", d)
		defer s.CloseWithError(0x1337, "Test finished!")
	}

	ticker := exponentialTicker(client.RequestsPerSec)
	done := make(chan bool)
	counter := make(chan int64, client.MaxConcurrentConnections)

	client.setupTestDuration(done)
	util.SetupGracefulShutdown(func() {
		done <- true
	})

	for {
		if len(counter) == cap(counter) {
			fmt.Printf("maxConcurrentConnections=%d reached\n", client.MaxConcurrentConnections)
			continue
		}
		id := time.Now().UnixNano()
		counter <- id
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.newReq(sessions, id); err != nil {
				return
			}
			<-counter
		}()

		select {
		case <-done:
			ticker.Stop()
			break
		case <-ticker.C:
		}
	}
	fmt.Printf("Waiting %d connections...\n", len(counter))
	wg.Wait()
	fmt.Println("Stopped")

	return nil
}

// newReq creates and sends a new request to the destinations
func (client *Client) newReq(session []quic.EarlyConnection, id int64) error {
	idReq := client.IdDevice + "_" + strconv.FormatInt(id, 10)
	req := util.RoPEMessage{
		ReqID: idReq,
		Log:   client.LoggerEnabled,
		Type:  util.Request,
	}

	// Determine the destination for the request
	dest := ForwardDecision(&req, client.Destinations)
	idxdest := 0
	for i, d := range client.Destinations {
		if d == dest {
			idxdest = i
			break
		}
	}

	// Open a new stream to the destination
	stream, err := session[idxdest].OpenStream()
	if err != nil {
		return err
	}

	// Encode and send the request
	encoder := gob.NewEncoder(stream)
	if err := encoder.Encode(req); err != nil {
		fmt.Printf("Error sending: %s\n", idReq)
		return err
	}

	// Decode and handle the response
	var packet util.RoPEMessage
	var rcv bool
	decoder := gob.NewDecoder(stream)
	var wg sync.WaitGroup
	for {
		if err := decoder.Decode(&packet); err == io.EOF || err != nil {
			break
		}
		rcv = true
		wg.Add(1)
		go forwardResponse(packet, &wg)
	}
	wg.Wait()

	if !rcv {
		fmt.Printf("Timeout or broken: %s\n", idReq)
		return err
	}

	return stream.Close()
}

// forwardResponse handles the received response asynchronously
func forwardResponse(packet util.RoPEMessage, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("Response received: type:", packet.Type)
	fmt.Println("Response received: ID:", packet.ReqID)
	fmt.Println("Response received: source:", packet.Source)
	if ForwardSetLastResponse != nil {
		ForwardSetLastResponse(packet)
	}
}
