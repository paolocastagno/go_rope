package routing

import (
	"context"
	"crypto/tls"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/paolocastagno/go_rope/pkg/config"
	"github.com/paolocastagno/go_rope/pkg/util"

	"github.com/pelletier/go-toml"

	"github.com/quic-go/quic-go"
)

var tokensUp, tokensDn chan struct{}

type Proxy struct {
	IdDevice     string
	ListenAddr   string
	ConfFile     string
	Timeout      time.Duration
	ParallelConn uint
	RoutingTbl   map[string]string
}

func (proxy *Proxy) InitProxy(configFile string, quicConf *quic.Config, initLogic func(*toml.Tree)) error {

	proxy.printParams()
	fmt.Printf("Starting idDevice: %s\n", proxy.IdDevice)

	util.SetupGracefulShutdown(func() {
		util.CloseLogger()
		os.Exit(0)
	})

	// Limit parallelism
	if proxy.ParallelConn > 0 {
		tokensUp = make(chan struct{}, proxy.ParallelConn)
		tokensDn = make(chan struct{}, proxy.ParallelConn)
	}

	//var RTT = make(map[string]float64)
	// Load forwarding logic
	if proxy.ConfFile != "" {
		fmt.Println("Loading forwarding policy:", proxy.ConfFile)
		//loadForwardingConf(proxy.ConfFile, RTT)
		config.LoadForwardingConf(proxy.ConfFile, initLogic)
	} else {
		config.Die("No forwarding policy specified")
	}

	log.Fatal(proxy.proxyMain(proxy.ListenAddr, quicConf))
	// Wait until all the ping tests are done
	//wgPing.Wait()

	return nil
}

func (proxy *Proxy) printParams() {
	for key, value := range proxy.RoutingTbl {
		fmt.Println("Destination: ", key, " Next hop: ", value)
	}
	fmt.Println(proxy.IdDevice)
	fmt.Println(proxy.ConfFile)
	fmt.Println(proxy.ListenAddr)
	fmt.Println(proxy.Timeout)
}

func (proxy *Proxy) proxyMain(inAddr string, quicConf *quic.Config) error {

	tlsConf := &tls.Config{ ////
		InsecureSkipVerify: true,
		NextProtos:         []string{"RoPEProtocol"},
	}

	ctx := context.Background()
	// ctx = context.WithValue(ctx, "source", idDevice)

	// Open connections towards different destinations
	var sessions = make(map[string]quic.Connection)
	for _, addr := range proxy.RoutingTbl {
		s, err := quic.DialAddrEarly(ctx, addr, tlsConf, quicConf)
		sessions[addr] = s
		if err != nil {
			fmt.Printf("Cannot connect to %s \n", addr)
			log.Println(err)
			return err
		}
		defer s.CloseWithError(0x1337, "Proxy finished")
	}

	// Wait for incoming connections
	listener, err := quic.ListenAddrEarly(inAddr, util.GenerateTLSConfig(), quicConf)
	if err != nil {
		return err
	}

	fmt.Printf("GO Proxy Ready %s\n", inAddr)
	fmt.Printf("Requests timeout set to %v\n", proxy.Timeout)

	for {
		// Accept incoming connection at in_addr
		sess, err := listener.Accept(context.Background())
		if err != nil {
			continue
		}

		// Handle a new connection in a go routine
		go proxy.newClient(sess, sessions)
	}
}

// Lisen for traffic incoming from a connection
func (proxy *Proxy) newClient(sess quic.Connection, sessions map[string]quic.Connection) {
	fmt.Printf("New connection from %s\n", sess.RemoteAddr().String())

	for {
		// Limit parallelism
		if proxy.ParallelConn > 0 {
			tokensUp <- struct{}{}
		}
		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			fmt.Println("AcceptStream error:", err)
			return
		}

		// Incoming requests are processed in a working thread
		go proxy.newRequest(stream, sessions)

	}
}

func (proxy *Proxy) newRequest(stream quic.Stream, sessions map[string]quic.Connection) {
	defer func(stream quic.Stream) {
		err := stream.Close()
		if err != nil {

		}
	}(stream)

	// Read the request
	var packet util.RoPEMessage
	decoder := gob.NewDecoder(stream)
	err := decoder.Decode(&packet)

	fmt.Println("\nin proxy:\ndest: ", packet.Destination)
	fmt.Println("type: ", packet.Type)
	fmt.Println("ID: ", packet.ReqID)
	//fmt.Println("source: ", packet.Source)

	if err != nil {
		return
	}

	util.LogEvent(packet.ReqID, util.Received, "Ricevuta richiesta proxy", packet.Log, proxy.IdDevice)

	// Create a response
	var resp util.RoPEMessage

	// Forwarding logic (application logic)
	if proxy.ConfFile != "" {
		ForwardDecision(&packet)
	}

	dest, ok := proxy.RoutingTbl[packet.Destination] ///
	if ok {
		// Send the response to the intended destination (the server)
		resp = proxy.forwardRequest(sessions[dest], packet)
	} else {
		// Destination not found
		//util.LogEvent(packet.ReqID, util.RoutingErr, "No route", packet.Log, idDevice)
		fmt.Println("\nNo route for ", packet.Destination, ". Available routes: ", dest)
		packet.Type = util.NoRoute
		packet.Body = make([]byte, 0)
		encoder := gob.NewEncoder(stream)
		_ = encoder.Encode(packet)

		//util.LogEvent(packet.ReqID, util.Sent, "Answering No route", packet.Log, idDevice)
		return
	}
	/* Log traffic
	if ForwardSetLastResponse != nil {
		ForwardSetLastResponse(&packet)
	}

	util.LogEvent(resp.ReqID, util.Sent, "Sending response", resp.Log, idDevice)
	util.LogEvent(resp.ReqID, util.Received, packet.Destination, resp.Log, idDevice)*/

	// Forward packet to the inended destination (the client who started the connection)
	encoder := gob.NewEncoder(stream)
	err = encoder.Encode(resp)
	if err != nil {
		return
	}

}

func (proxy *Proxy) forwardRequest(serverSession quic.Connection, packet util.RoPEMessage) util.RoPEMessage {

	errorResp := util.RoPEMessage{ReqID: packet.ReqID,
		Log:         packet.Log,
		ResSize:     packet.ResSize,
		Type:        util.ServerNotFound,
		Body:        make([]byte, 0),
		Source:      packet.Source,
		Hop:         proxy.IdDevice,
		Destination: packet.Destination}

	// Forward the packet
	stream, err := serverSession.OpenStream() //OpenStreamSync(context.Background())
	if err != nil {
		return errorResp
	}
	defer func(stream quic.Stream) {
		err := stream.Close()
		if err != nil {

		}
	}(stream)

	encoder := gob.NewEncoder(stream)
	err = encoder.Encode(packet)

	util.LogEvent(packet.ReqID, util.Sent, packet.Destination, packet.Log, proxy.IdDevice)

	if proxy.ParallelConn > 0 {
		<-tokensUp
	}

	if err != nil {
		fmt.Printf("Error forwarding: %s\n", packet.ReqID)
		return errorResp
	}

	// RICEVE LA RISPOSTA DAL SERVER (MEC O CLOUD)
	if proxy.ParallelConn > 0 {
		tokensDn <- struct{}{}
	}
	var resp util.RoPEMessage
	decoder := gob.NewDecoder(stream)
	err = decoder.Decode(&resp)

	if err != nil {
		fmt.Printf("Timeout or broken: %s\n", packet.ReqID)
		errorResp.Type = util.ServerTimeout
		util.LogEvent(packet.ReqID, util.Timeout, packet.Destination, packet.Log, proxy.IdDevice)
		return errorResp
	}

	if ForwardSetLastResponse != nil {
		ForwardSetLastResponse(&resp)
	}

	// util.LogEvent(resp.ReqID, util.Received, packet.Destination, resp.Log, idDevice)
	util.LogEvent(resp.ReqID, util.Sent, "Sending response", resp.Log, proxy.IdDevice)

	if proxy.ParallelConn > 0 {
		<-tokensDn
	}

	return resp
}
