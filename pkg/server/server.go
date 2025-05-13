package server

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/paolocastagno/go_rope/pkg/util"

	"github.com/pelletier/go-toml"
	"github.com/quic-go/quic-go"
)

var GitCommit = "master"

// JobRequest represents a job request containing a RoPE message and a QUIC stream
type JobRequest struct {
	Request    util.RoPEMessage
	QuicStream quic.Stream
}

// sessions stores active QUIC connections
var sessions map[string]quic.EarlyConnection

var ForwardBlock func(*util.RoPEMessage, quic.Stream)
var ForwardDecision func(*util.RoPEMessage, *map[string]quic.EarlyConnection, int64) bool
var ForwardSetLastResponse func(*util.RoPEMessage)

// Server represents the server configuration
type Server struct {
	IdDevice       string
	ListenAddr     string
	BufSize        int64
	Workers        int64
	ProcessingTime time.Duration
	Timeout        time.Duration
	Application    *interface{}
	LoggerEnabled  bool
}

// InitServer initializes the server with the given QUIC configuration and forwarding logic
func InitServer(server *Server, quicConf *quic.Config, initLogic func(*toml.Tree) *interface{}, CfgFile string) error {
	fmt.Printf("Running server version %s\n", GitCommit)

	// Load and parse the configuration file
	cfg, err := loadConfig(CfgFile)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %v", err)
	}

	// Parse server configuration
	if err := parseServerConfig(server, cfg); err != nil {
		return fmt.Errorf("failed to parse server configuration: %v", err)
	}

	// Initialize application logic
	applicationConfig, ok := cfg.Get("application").(*toml.Tree)
	if !ok {
		return errors.New("missing or invalid 'application' section in configuration")
	}
	server.Application = initLogic(applicationConfig)

	// Initialize logger if enabled
	if server.LoggerEnabled {
		if err := initializeLogger(cfg); err != nil {
			return fmt.Errorf("failed to initialize logger: %v", err)
		}
	}

	fmt.Printf("Starting server idDevice: %s\n", server.IdDevice)

	// Setup graceful shutdown
	util.SetupGracefulShutdown(func() {
		util.CloseLogger()
		os.Exit(0)
	})

	return nil
}

// loadConfig loads the TOML configuration file
func loadConfig(CfgFile string) (*toml.Tree, error) {
	cfg, err := toml.LoadFile(CfgFile)
	if err != nil {
		return nil, fmt.Errorf("error loading configuration file: %v", err)
	} else {
		fmt.Printf("Configuration file loaded: %s\n", cfg.String())
	}
	return cfg, nil
}

// parseServerConfig parses the server-specific configuration
func parseServerConfig(server *Server, cfg *toml.Tree) error {
	var err error

	if idDevice, ok := cfg.Get("configuration.IdDevice").(string); ok {
		server.IdDevice = idDevice
	} else {
		return errors.New("missing or invalid 'IdDevice' in configuration")
	}

	if listenAddr, ok := cfg.Get("configuration.ListenAddress").(string); ok {
		server.ListenAddr = listenAddr
	} else {
		return errors.New("missing or invalid 'ListenAddress' in configuration")
	}

	if bufSize, ok := cfg.Get("configuration.BufSize").(int64); ok {
		server.BufSize = bufSize
	} else {
		return errors.New("missing or invalid 'BufSize' in configuration[" + fmt.Sprintf("%d", bufSize) + "]")
	}

	if workers, ok := cfg.Get("configuration.Workers").(int64); ok {
		server.Workers = workers
	} else {
		return errors.New("missing or invalid 'Workers' in configuration")
	}

	if processingTime, ok := cfg.Get("configuration.ProcessingTime").(string); ok {
		server.ProcessingTime, err = time.ParseDuration(processingTime)
		if err != nil {
			return fmt.Errorf("error parsing 'ProcessingTime': %v", err)
		}
	} else {
		return errors.New("missing or invalid 'ProcessingTime' in configuration")
	}

	if timeout, ok := cfg.Get("configuration.Timeout").(string); ok {
		server.Timeout, err = time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("error parsing 'Timeout': %v", err)
		}
	} else {
		return errors.New("missing or invalid 'Timeout' in configuration")
	}

	return nil
}

// initializeLogger initializes the logger based on the configuration
func initializeLogger(cfg *toml.Tree) error {
	loggerCfg, ok := cfg.Get("logger").(*toml.Tree)
	if !ok {
		return errors.New("missing or invalid 'logger' section in configuration")
	}
	util.SetLoggerParamFromConf(loggerCfg)
	return nil
}

// run starts the server and handles incoming connections
func Run(server *Server, quicConf *quic.Config, ForwardDecision func(*util.RoPEMessage, *map[string]quic.EarlyConnection, int64) bool, ForwardSetLastResponse func(*util.RoPEMessage), ForwardBlock func(*util.RoPEMessage, quic.Stream)) error {
	requestQueue := make(chan JobRequest, server.BufSize)
	listener, err := quic.ListenAddrEarly(server.ListenAddr, util.GenerateTLSConfig(), quicConf)
	if err != nil {
		return fmt.Errorf("error creating listener: %v", err)
	}
	defer listener.Close()

	// Modernized loop using range over a slice of integers
	for i := range make([]int, server.Workers) {
		go worker(i, requestQueue, ForwardDecision, ForwardSetLastResponse)
	}

	fmt.Printf("Server ready %s, workers %d, queue length=%d\n", server.ListenAddr, server.Workers, server.BufSize)

	for {
		sess, err := listener.Accept(context.Background())
		if err != nil {
			fmt.Printf("Session error: %s\n", err)
			return err
		}
		go newRequest(sess, requestQueue, ForwardBlock)
	}
}

// newRequest handles new incoming QUIC connections and streams
func newRequest(sess quic.Connection, queue chan<- JobRequest, ForwardBlock func(*util.RoPEMessage, quic.Stream)) {
	for {
		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			fmt.Println("Accept Stream error:", err)
			if err.Error() == "Application error 0x1337: Finish" {
				os.Exit(0)
			}
			return
		}
		go newRequestServer(stream, queue, ForwardBlock)
	}
}

// newRequestServer processes a new request and adds it to the job queue
func newRequestServer(stream quic.Stream, queue chan<- JobRequest, ForwardBlock func(*util.RoPEMessage, quic.Stream)) {
	var packet util.RoPEMessage
	decoder := gob.NewDecoder(stream)
	err := decoder.Decode(&packet)
	if err != nil {
		fmt.Printf("Request error: %s\n", err)
		return
	}

	select {
	case queue <- JobRequest{packet, stream}:
	default:
		fmt.Printf("Queue full\n")
		handleQueueFull(packet, stream, ForwardBlock)
	}
}

// handleQueueFull handles the case when the job queue is full
func handleQueueFull(packet util.RoPEMessage, stream quic.Stream, ForwardBlock func(*util.RoPEMessage, quic.Stream)) {
	if ForwardBlock != nil {
		ForwardBlock(&packet, stream)
	} else {
		packet.Type = util.QueueFull
		packet.Body = make([]byte, 0)
		packet.Hop = packet.Destination
		packet.Destination = packet.Source
		packet.Source = packet.Hop
		encoder := gob.NewEncoder(stream)
		err := encoder.Encode(packet)
		if err != nil {
			fmt.Printf("Error while sending 'Queue full' message: %s \n", err)
		}
	}
}

// worker processes job requests from the queue
func worker(i int, queue <-chan JobRequest, ForwardDecision func(*util.RoPEMessage, *map[string]quic.EarlyConnection, int64) bool, ForwardSetLastResponse func(*util.RoPEMessage)) {
	for request := range queue {
		packet := request.Request
		stream := request.QuicStream

		fmt.Printf("Worker %d processing request from queue; Queue length [%d]\n", i, len(queue))

		// packet.Body = make([]byte, packet.ResSize)
		if int64(len(packet.Body)) < int64(packet.ResSize) {
			padding := make([]byte, int64(packet.ResSize)-int64(len(packet.Body)))
			packet.Body = append(packet.Body, padding...)
		} else if int64(len(packet.Body)) > int64(packet.ResSize) {
			packet.Body = packet.Body[:packet.ResSize]
		}
		packet.Type = util.Response

		send := true
		var attempt int64 = 0
		for send {
			send = ForwardDecision(&packet, &sessions, attempt)
			attempt++
			if packet.Destination != " " {
				encoder := gob.NewEncoder(stream)
				err := encoder.Encode(packet)
				if err != nil {
					fmt.Printf("Error sending response: %s \n", err)
				}
				if ForwardSetLastResponse != nil {
					ForwardSetLastResponse(&packet)
				}
			}
		}
		err := stream.Close()
		if err != nil {
			return
		}
	}
}
