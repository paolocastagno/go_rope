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

/*type ServerConfig struct {
	IdDevice    string
	Proxy       string
	ServiceTime time.Duration
	Port        string
	QueueLen    uint
	Workers     uint
	Logger      util.LoggerConf
}*/

type destAddresses []string

func (i *destAddresses) String() string {
	return "my string representation"
}

func (i *destAddresses) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type JobRequest struct {
	Request    util.RoPEMessage
	QuicStream quic.Stream
}

var sessions map[string]quic.EarlyConnection

type Server struct {
	IdDevice   string
	ListenAddr string
	//Destinations destAddresses //se == listenaddr mi fermo
	AppCfg   string
	QueueLen int
	Workers  int
	Timeout  time.Duration
}

func NewServer(idDevice string, listenAddr string, appCfg string, queueLen int, workers int, timeout time.Duration) *Server {
	return &Server{
		IdDevice:   idDevice,
		ListenAddr: listenAddr,
		//Destinations: destinations,
		AppCfg:   appCfg,
		QueueLen: queueLen,
		Workers:  workers,
		Timeout:  timeout,
	}
}

func (server *Server) InitServer(quicConf *quic.Config, initLogic func(*toml.Tree)) error {
	// Configure application
	if server.AppCfg != "" {
		fmt.Println("Loading forwaring policy:", server.AppCfg)
		// loadForwardingConf(server.AppCfg, server.Destinations, RTT)
		//loadForwardingConf(server.AppCfg /*server.Destinations*/)
		config.LoadForwardingConf(server.AppCfg, initLogic)
	} else {
		return errors.New("no forwarding policy specified")
	}

	// Initialize logger
	if err := util.InitLogger(); err != nil {
		return fmt.Errorf("error initializing logger: %v", err)
	}
	defer util.CloseLogger()

	// Handle graceful shutdown
	util.SetupGracefulShutdown(func() {
		util.CloseLogger()
		os.Exit(0)
	})

	// Wait until all the ping tests are done
	//wgPing.Wait()

	// Start the server
	return server.run(quicConf)
}

func (server *Server) run(quicConf *quic.Config) error {

	requestQueue := make(chan JobRequest, server.QueueLen)
	listener, err := quic.ListenAddrEarly(server.ListenAddr, util.GenerateTLSConfig(), quicConf)
	if err != nil {
		fmt.Printf("Error in creating listener: %s\n", err)
	}

	time := time.Now().UnixNano()
	// Create and launch go routines to handle incoming requests
	for i := 0; i < server.Workers; i++ {
		go worker(i, requestQueue, time)
	}

	fmt.Printf("Server ready %s, workers %d, queue length=%d\n", server.ListenAddr, server.Workers, server.QueueLen)

	for { ///////CICLO PRINCIPALE SERVER
		sess, err := listener.Accept(context.Background())
		fmt.Println("accettato nuovo listener")
		if err != nil {
			fmt.Printf("Session error: %s\n", err)
			return err
		}

		go newRequest(sess, requestQueue)

	}
}

// newRequest handles an incoming stream
// func newRequest(sess quic.Session, queue chan<- JobRequest) {
func newRequest(sess quic.Connection, queue chan<- JobRequest) {
	fmt.Println("Preparazione dell'elaborazione delle richieste")
	for {
		// Session's stream
		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			fmt.Println("Accept Stream error:", err)
			fmt.Println(err.Error())
			if err.Error() == "Application error 0x1337: Finish" {
				os.Exit(0) // Terminates the experiment
			}
			return
		}
		// Launch a new go routine to process a request and send a response
		go newRequestServer(stream, queue)
	}
}

// newRequestServer receives and enqueue incoming packets
func newRequestServer(stream quic.Stream, queue chan<- JobRequest) {
	var packet util.RoPEMessage
	// Read the incoming request
	decoder := gob.NewDecoder(stream)
	err := decoder.Decode(&packet)
	fmt.Printf("\nREAD INCOMING REQUEST\n")
	fmt.Printf("Source: %s, Destination: %s\n", packet.ReqID, packet.Destination)

	if err != nil {
		fmt.Printf("Request error: %s\n", err)
	}
	//util.LogEvent(packet.ReqID, util.Received, "Recieved packet", packet.Log, idDevice)
	select {
	case queue <- JobRequest{packet, stream}:
		//util.LogEvent(packet.ReqID, util.Enqueued, fmt.Sprintf("Queuing packet; Queue occupacy [%d]", len(queue)), packet.Log, idDevice)

	default:
		fmt.Printf("Queue full\n")
		//util.LogEvent(packet.ReqID, util.FullQueue, "Queue full", packet.Log, idDevice)
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
			//util.LogEvent(packet.ReqID, util.Sent, "Sending 'Queue full'", packet.Log, idDevice)
			if err != nil {
				fmt.Printf("Error while sending 'Queue full' message: %s \n", err)
			}
		}
	}

}

// worker dequeue a packet, process it, and sends a response
func worker(i int, queue <-chan JobRequest, t int64) {
	//viene ritornato la JobRequest all'interno della queue
	for request := range queue {
		packet := request.Request    //pacchetto di tipo Message
		stream := request.QuicStream //Ricava lo stream

		fmt.Printf("\nSENDING RESPONSE\n")
		fmt.Printf("Worker %d preleva richiesta dalla coda; Occupazione [%d]\n", i, len(queue))
		//util.LogEvent(packet.ReqID, util.Dequeued, fmt.Sprintf("Worker %d preleva richiesta dalla coda; Occupazione [%d]", i, len(queue)), packet.Log, idDevice)

		//Generate a response
		packet.Body = make([]byte, packet.ResSize)
		packet.Type = util.Response ////
		// Call the application logic
		send := true
		var i int64 = 0
		for send {
			send = ForwardDecision(&packet, &sessions, i)
			i += 1
			//util.LogEvent(packet.ReqID, util.Processed, fmt.Sprintf("Worker %d termina elaborazione", i), packet.Log, idDevice)
			//fmt.Println("pre send risposta", packet.Destination)

			// Send a response
			if packet.Destination != " " { //" "
				encoder := gob.NewEncoder(stream)
				err := encoder.Encode(packet)
				//util.LogEvent(packet.ReqID, util.Sent, fmt.Sprintf("Worker %d invia risposta", i), packet.Log, idDevice)
				fmt.Printf("SENT\n\n")

				if ForwardSetLastResponse != nil {
					ForwardSetLastResponse(&packet)
				}

				if err != nil {
					fmt.Printf("Errore nell'invio della risposta %s \n", err)
				}
			}

		}
		err := stream.Close()
		if err != nil {
			return
		}

	}
}
