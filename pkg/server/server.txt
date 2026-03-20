package server

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/paolocastagno/go_rope/pkg/config"
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
var (
	sessions                    map[string]quic.EarlyConnection
	ForwardDecision             func(*util.RoPEMessage, *map[string]quic.EarlyConnection, int64) bool
	ForwardBlock                func(*util.RoPEMessage, quic.Stream)
	ForwardSetLastResponse      func(*util.RoPEMessage)
	ForwardSetLastResponseBlock func(*util.RoPEMessage, quic.Stream)
)

// Server represents the server configuration
type Server struct {
	IdDevice   string
	ListenAddr string
	AppCfg     string
	QueueLen   int
	Workers    int
	Timeout    time.Duration
}

// NewServer creates a new Server instance with the given parameters
func NewServer(idDevice string, listenAddr string, appCfg string, queueLen int, workers int, timeout time.Duration) *Server {
	return &Server{
		IdDevice:   idDevice,
		ListenAddr: listenAddr,
		AppCfg:     appCfg,
		QueueLen:   queueLen,
		Workers:    workers,
		Timeout:    timeout,
	}
}

// InitServer initializes the server with the given QUIC configuration and forwarding logic
func (server *Server) InitServer(quicConf *quic.Config, initLogic func(*toml.Tree)) error {
	fmt.Printf("Running server version %s\n", GitCommit)

	// Initialize logger
	if err := util.InitLogger(); err != nil {
		return fmt.Errorf("error initializing logger: %v", err)
	}
	defer util.CloseLogger()

	fmt.Printf("Starting server idDevice: %s\n", server.IdDevice)

	// Load forwarding policy
	if server.AppCfg != "" {
		fmt.Println("Loading forwarding policy:", server.AppCfg)
		config.LoadForwardingConf(server.AppCfg, initLogic)
	} else {
		return errors.New("no forwarding policy specified")
	}

	// Setup graceful shutdown
	util.SetupGracefulShutdown(func() {
		util.CloseLogger()
		os.Exit(0)
	})

	return server.run(quicConf)
}

// run starts the server and handles incoming connections
func (server *Server) run(quicConf *quic.Config) error {
	requestQueue := make(chan JobRequest, server.QueueLen)
	listener, err := quic.ListenAddrEarly(server.ListenAddr, util.GenerateTLSConfig(), quicConf)
	if err != nil {
		return fmt.Errorf("error creating listener: %v", err)
	}
	defer listener.Close()

	for i := 0; i < server.Workers; i++ {
		go worker(i, requestQueue)
	}

	fmt.Printf("Server ready %s, workers %d, queue length=%d\n", server.ListenAddr, server.Workers, server.QueueLen)

	for {
		sess, err := listener.Accept(context.Background())
		if err != nil {
			fmt.Printf("Session error: %s\n", err)
			return err
		}
		go newRequest(sess, requestQueue)
	}
}

// newRequest handles new incoming QUIC connections and streams
func newRequest(sess quic.Connection, queue chan<- JobRequest) {
	for {
		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			fmt.Println("Accept Stream error:", err)
			if err.Error() == "Application error 0x1337: Finish" {
				os.Exit(0)
			}
			return
		}
		go newRequestServer(stream, queue)
	}
}

// newRequestServer processes a new request and adds it to the job queue
func newRequestServer(stream quic.Stream, queue chan<- JobRequest) {
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
		handleQueueFull(packet, stream)
	}
}

// handleQueueFull handles the case when the job queue is full
func handleQueueFull(packet util.RoPEMessage, stream quic.Stream) {
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
func worker(i int, queue <-chan JobRequest) {
	for request := range queue {
		packet := request.Request
		stream := request.QuicStream

		fmt.Printf("Worker %d processing request from queue; Queue length [%d]\n", i, len(queue))

		packet.Body = make([]byte, packet.ResSize)
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
