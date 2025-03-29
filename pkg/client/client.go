package client

import (
	"context"
	"crypto/tls"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/pelletier/go-toml"
	"github.com/quic-go/quic-go"

	"github.com/paolocastagno/go_rope/pkg/util"
)

var GitCommit = "master"

// ForwardDecision is a function that decides the forwarding logic for a message
var ForwardDecision func(msg *util.RoPEMessage, destinations []string) string

// ForwardSetLastResponse is a function that sets the last response for a message
var ForwardSetLastResponse func(util.RoPEMessage)

// ApplicationLogic defines the interface for application logic
type ApplicationLogic interface {
	InitApp(*toml.Tree)
	App(*util.RoPEMessage, []string) string
	pktSnd(*util.RoPEMessage)
}

// Client represents the client configuration
type Client struct {
	IdDevice                 string
	Destinations             []string
	MaxConcurrentConnections uint
	TestDuration             time.Duration
	Timeout                  time.Duration
	CfgFile                  string
	LoggerEnabled            bool
	Connections              []quic.EarlyConnection
	Counter                  chan int64
	Application              ApplicationLogic
}

// InitClient initializes the client with the given configuration file and QUIC configuration
func (client *Client) InitClient(quicConf *quic.Config, initLogic func(*toml.Tree)) error {
	fmt.Printf("Running client version %s\n", GitCommit)

	// Load client configuration file
	if client.CfgFile != "" {
		config, err := toml.LoadFile(client.CfgFile)
		if err != nil {
			return fmt.Errorf("error loading configuration: %v", err)
		} else {
			initLogic(config)
		}
		fmt.Println("Loading forwarding policy:", client.CfgFile)
		client.Application.InitApp(config)
	} else {
		return errors.New("no forwarding policy specified")
	}
	fmt.Printf("Starting idDevice: %s\n", client.IdDevice)

	// Initialize the logger
	if err := util.InitLogger(); err != nil {
		panic(err)
	}
	defer util.CloseLogger()

	client.LoggerEnabled = util.IsLoggerEnabled()

	// Start the client main process
	if err := client.clientMain(client.Destinations, quicConf); err != nil {
		fmt.Println("Error!", err)
	}

	fmt.Println("Finished! Waiting 20 seconds...")
	time.Sleep(20 * time.Second)

	return nil
}

// GetString retrieves a string value from the configuration map
func GetString(cfg map[string]interface{}, key, defaultValue string) string {
	if value, ok := cfg[key].(string); ok {
		return value
	}
	return defaultValue
}

// GetFloat64 retrieves a float64 value from the configuration map
func GetFloat64(cfg map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := cfg[key].(float64); ok {
		return value
	}
	return defaultValue
}

// GetDuration retrieves a duration value from the configuration map
func GetDuration(cfg map[string]interface{}, key, defaultValue string) time.Duration {
	if value, ok := cfg[key].(string); ok {
		duration, err := time.ParseDuration(value)
		if err == nil {
			return duration
		}
	}
	duration, _ := time.ParseDuration(defaultValue)
	return duration
}

// PrintParams prints the client parameters
func (client *Client) PrintParams() {
	fmt.Println("Test duration:", client.TestDuration)
	fmt.Println("Timeout:", client.Timeout)
	fmt.Println("Destinations:", client.Destinations)
	fmt.Println("IdDevice:", client.IdDevice)
	fmt.Println("MaxConcurrentConnections:", client.MaxConcurrentConnections)
	fmt.Println("CfgFile:", client.CfgFile)
}

// SetupTestDuration sets up the test duration timer
func (client *Client) SetupTestDuration(done chan<- bool) {
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

// clientMain configures the QUIC connections with the destinations
func (client *Client) clientMain(destinations []string, quicConf *quic.Config) error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"RoPEProtocol"},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "source", client.IdDevice)

	fmt.Printf("Requests timeout set to %v\n", client.Timeout)

	for _, d := range destinations {
		c, err := quic.DialAddrEarly(ctx, d, tlsConf, quicConf)
		if err != nil {
			fmt.Printf("Cannot connect to %s\n", d)
			log.Println(err)
			return err
		}
		client.Connections = append(client.Connections, c)
		fmt.Printf("Connected to %s\n", d)
	}

	return nil
}

func (client *Client) close() {
	for _, s := range client.Connections {
		s.CloseWithError(0x1337, "Closing client's connection")
	}
}

// NewReq creates and sends a new request to the destinations
func (client *Client) NewReq(session []quic.EarlyConnection, id int64) error {
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
