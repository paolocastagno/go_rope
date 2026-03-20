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
var ForwardDecision func(msg *util.RoPEMessage, destinations []string, application *interface{}) string

// ForwardSetLastResponse is a function that sets the last response for a message
var ForwardSetLastResponse func(util.RoPEMessage, *interface{})

// Client represents the client configuration
type Client struct {
	IdDevice                 string
	Destinations             []string
	MaxConcurrentConnections uint
	Done                     chan bool
	TestDuration             time.Duration
	Timeout                  time.Duration
	CfgFile                  string
	LoggerEnabled            bool
	Connections              []quic.EarlyConnection
	Counter                  chan int64
	Application              *interface{} // Pointer to an interface
}

// InitClient initializes the client with the given configuration file and QUIC configuration
func (client *Client) InitClient(quicConf *quic.Config, CfgFile string, initLogic func(*toml.Tree) *interface{}) error {
	fmt.Printf("Running client version %s\n", GitCommit)
	client.CfgFile = CfgFile

	// Initialize the Done channel
	client.Done = make(chan bool)

	// Load and parse the configuration file
	err := loadConfig(client, initLogic)
	if err != nil {
		return err
	}

	// Set up the test duration if defined
	client.setupTestDuration()

	// Start the client main process
	if err := client.clientMain(client.Destinations, quicConf); err != nil {
		fmt.Println("Error!", err)
	}

	return nil
}

// Helper function to load the configuration file
func loadConfig(client *Client, initLogic func(*toml.Tree) *interface{}) error {
	if client.CfgFile == "" {
		return errors.New("no configuration file specified")
	}

	config, err := toml.LoadFile(client.CfgFile)
	if err != nil {
		return fmt.Errorf("error loading configuration: %v", err)
	}

	cfgMap := config.ToMap()
	fmt.Printf("Config contents: %v\n", cfgMap)

	// Safely retrieve the "configuration" section
	configSection, ok := cfgMap["configuration"]
	if !ok {
		return errors.New("missing 'configuration' section in config file")
	}
	configMap, ok := configSection.(map[string]interface{})
	if !ok {
		return fmt.Errorf("'configuration' section must be a table, got %T", configSection)
	}

	client.IdDevice = GetString(configMap, "id_device", "default_id")

	// Safely retrieve the destinations
	configDestinations, ok := config.Get("configuration.destinations").([]interface{})
	if !ok || len(configDestinations) == 0 {
		return errors.New("missing or invalid 'destinations' key in 'configuration' section")
	}

	// Convert []interface{} to []string
	client.Destinations = make([]string, len(configDestinations))
	for i, d := range configDestinations {
		str, ok := d.(string)
		if !ok {
			return fmt.Errorf("invalid destination value at index %d: expected string, got %T", i, d)
		}
		client.Destinations[i] = str
	}

	// Safely get MaxConcurrentConnections
	maxConnRaw := config.GetDefault("configuration.MaxConcurrentConnections", int64(1))
	maxConn, ok := maxConnRaw.(int64)
	if !ok {
		return fmt.Errorf("MaxConcurrentConnections must be an integer, got %T", maxConnRaw)
	}
	if maxConn <= 0 {
		return fmt.Errorf("MaxConcurrentConnections must be positive, got %d", maxConn)
	}
	client.MaxConcurrentConnections = uint(maxConn)

	client.TestDuration = GetDuration(configMap, "TestDuration", "0s")
	client.Timeout = GetDuration(configMap, "timeout", "30s")
	client.Counter = make(chan int64, client.MaxConcurrentConnections)
	client.LoggerEnabled = config.GetDefault("logger_enabled", false).(bool)
	client.Connections = make([]quic.EarlyConnection, 0, client.MaxConcurrentConnections)

	// Retrieve the application section
	applicationConfig, ok := config.Get("application").(*toml.Tree)
	if !ok {
		return errors.New("missing or invalid 'application' section in configuration")
	}

	client.Application = initLogic(applicationConfig)

	if client.LoggerEnabled {
		loggercfg, ok := config.Get("logger").(*toml.Tree)
		if !ok {
			return errors.New("no logger configuration specified")
		}
		util.SetLoggerParamFromConf(loggercfg)
	}
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

// setupTestDuration sets up the test duration timer
func (client *Client) setupTestDuration() {
	if client.TestDuration > 0 {
		fmt.Printf("Test duration set to %v\n", client.TestDuration)
		testTimer := time.NewTimer(client.TestDuration)
		go func() {
			<-testTimer.C
			fmt.Println("Test ended")
			client.Done <- true
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
	dest := ForwardDecision(&req, client.Destinations, client.Application)
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
		go forwardResponse(packet, &wg, client.Application)
	}
	wg.Wait()

	if !rcv {
		fmt.Printf("Timeout or broken: %s\n", idReq)
		return err
	}

	return stream.Close()
}

// forwardResponse handles the received response asynchronously
func forwardResponse(packet util.RoPEMessage, wg *sync.WaitGroup, app *interface{}) {
	defer wg.Done()
	// fmt.Println("Response received: type:", packet.Type)
	// fmt.Println("Response received: ID:", packet.ReqID)
	// fmt.Println("Response received: source:", packet.Source)
	if ForwardSetLastResponse != nil {
		ForwardSetLastResponse(packet, app)
	}
}
