package client

import (
	"go_rope/config"
	"go_rope/util"

	"github.com/pelletier/go-toml"

	//"context"
	"context"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"io"

	"github.com/lthibault/jitterbug"
	"github.com/quic-go/quic-go"
	"gonum.org/v1/gonum/stat/distuv"
)

var GitCommit = "master"

// ClientConfig is the configuration for the current client
// A configuration can be defined in a config file following
// the toml structure
// type ClientConfig struct {
// IdDevice						string			`json:"idDevice"`
// Proxy						string			`json:"proxy"`
// RequestsPerSec				uint			`json:"requestsPerSec"`
// MaxConcurrentConnections		uint			`json:"maxConcurrentConnections"`
// RequestSize					uint			`json:"requestSize"`
// ResponseSize					uint			`json:"responseSize"`
// TestDuration					string			`json:"testDuration"`
// Timeout						string			`json:"timeout"`
// Logger						util.LoggerConf	`json:"logger"`
// }

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

/*func NewClient(idDevice string, appCfg string, timeout time.Duration) *Client {
	return &Client{
		IdDevice:                 idDevice,
		Destinations:             destinations,
		RequestsPerSec:           requestsPerSec,
		MaxConcurrentConnections: maxConcurrentConnections,
		TestDuration:             testDuration,
		Timeout:                  timeout,
		Clicfg:                   clicfg,
		Appcfg:                   appCfg,
		LoggerEnabled:            false, //false per impostazione predefinita
		Sessions:                 nil,   //nil per impostazione predefinita
		Counter:                  make(chan int64, client.MaxConcurrentConnections),
	}
}*/

func (client *Client) InitClient(configFile string, quicConf *quic.Config, initLogic func(*toml.Tree)) error {

	fmt.Printf("Running client version %s\n", GitCommit)

	client.loadParam(configFile) //cfg :=

	fmt.Printf("Starting idDevice: %s\n", client.IdDevice)

	err := util.InitLogger()
	if err != nil {
		panic(err)
	}
	defer util.CloseLogger()

	client.LoggerEnabled = util.IsLoggerEnabled()

	var wgPing sync.WaitGroup

	// Configure application
	if client.Appcfg != "" {
		fmt.Println("Loading forwarding policy:", client.Appcfg)

		//LoadForwardingConf(client.Appcfg)
		config.LoadForwardingConf(client.Appcfg, initLogic) //working
		//config.LoadForwardingConf(client.Appcfg, client.Proxy) //omettere destination? non usato in impl //working with maps

	} else {
		return errors.New("no forwarding policy specified")
	}

	err = client.clientMain(client.Destinations, quicConf)
	if err != nil {
		//panic(err)
		fmt.Println("Error!", err)
	}

	fmt.Println("Finished! Waiting 20 seconds...")

	time.Sleep(20 * time.Second)

	wgPing.Wait()

	return nil
}

func (client *Client) loadParam(config string) bool {
	jsonFile, err := os.Open(config)
	if err == nil {
		fmt.Println("Using config file: ", config)
		defer func(jsonFile *os.File) {
			err := jsonFile.Close()
			if err != nil {

			}
		}(jsonFile)

		byteValue, _ := io.ReadAll(jsonFile)

		var cfg interface{}
		errj := json.Unmarshal(byteValue, &cfg)

		cfgMap := cfg.(map[string]interface{})

		fmt.Println(cfgMap)
		if errj != nil {
			fmt.Println("error:", errj)
		}

		if client.IdDevice == "none" {
			client.IdDevice = "device_default"
		} else {
			client.IdDevice = cfgMap["IdDevice"].(string)
		}

		//viene aggiunto solo l'indirizzo del proxy, la dest finale si trova nel msg
		client.Proxy = cfgMap["Proxy"].(string)
		client.Destinations = append(client.Destinations, client.Proxy)
		// requestInterval, _ = time.ParseDuration(defConf.RequestInterval)
		/*for _, v := range cfgMap["destinations"].([]interface{}) {
			client.Destinations = append(client.Destinations, fmt.Sprint(v))
			fmt.Println(client.Destinations)
		}*/
		client.RequestsPerSec = cfgMap["RequestsPerSec"].(float64)
		client.MaxConcurrentConnections = uint(cfgMap["MaxConcurrentConnections"].(float64))
		client.TestDuration, _ = time.ParseDuration(cfgMap["TestDuration"].(string))
		client.Timeout, _ = time.ParseDuration(cfgMap["Timeout"].(string))
		client.Appcfg = cfgMap["AppCfg"].(string)

		byteValue, err = json.Marshal(cfgMap["Logger"])
		if err == nil {
			var logger util.LoggerConf
			err := json.Unmarshal(byteValue, &logger)
			if err != nil {
				return false
			}
			util.SetLoggerParamFromConf(logger)
		}

		fmt.Printf("Client %s configuration:\n", client.IdDevice)
		client.printParams()
		return true
	} else {
		return false
	}
}

func (client *Client) printParams() {
	fmt.Println("Test duration: ", client.TestDuration)
	fmt.Println(client.Timeout)
	fmt.Println(client.Destinations)
	fmt.Println(client.IdDevice)
	fmt.Println(client.RequestsPerSec)
	fmt.Println(client.MaxConcurrentConnections)
	fmt.Println(client.TestDuration)
	fmt.Println(client.Timeout)
	fmt.Println(client.Appcfg)
}

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

func exponentialTicker(rps float64) *jitterbug.Ticker {

	// auto tune
	fmt.Println("Tuning value to get", rps, "request per second.")

	rand.Seed(time.Now().UTC().UnixNano())

	beta := 0.000000001 * float64(rps)
	// var beta float64 = 1.0 / rps

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

	var testSec uint = 20

	testTimer := time.NewTimer(time.Second * time.Duration(testSec))
	go func() {
		<-testTimer.C
		fmt.Println("Tuning ended")
		done <- true
	}()

	var counter float64 = 0

	start := time.Now()
external:
	for {
		//fmt.Println(counter)
		counter++

		select {
		case <-done:
			t.Stop()
			break external
		case <-t.C:
			continue external
		}
	}

	end := time.Now()

	fmt.Println("Duration:", end.Sub(start))

	fmt.Println("Expected ", rps*float64(testSec), "requests in", testSec, "seconds")
	fmt.Println("Got", counter, "requests in", testSec, "seconds")

	// newBeta := beta + (beta - counter/float64(testSec)*0.000000001)
	newBeta := beta

	fmt.Println("New beta:", newBeta)

	newTicker := jitterbug.New(
		time.Millisecond*0,
		&jitterbug.Univariate{
			Sampler: &distuv.Gamma{
				Alpha: 1,
				Beta:  newBeta,
			},
		},
	)

	return newTicker
	// return t
}

// Configura le connessioni QUIC con le destinazioni
func (client *Client) clientMain(destinations []string, quicConf *quic.Config) error {

	tlsConf := &tls.Config{ ///
		InsecureSkipVerify: true,
		NextProtos:         []string{"RoPEProtocol"},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "source", client.IdDevice)

	var wg sync.WaitGroup

	fmt.Printf("Requests per second set to %v\n", client.RequestsPerSec)
	fmt.Printf("Requests timeout set to %v\n", client.Timeout)
	var sessions []quic.EarlyConnection
	for i, d := range destinations {
		// connect to all destinations
		s, err := quic.DialAddrEarly(ctx, d, tlsConf, quicConf)
		sessions = append(sessions, s)
		// func DialAddrEarly(addr string, tlsConf *tls.Config, config *Config) (EarlySession, error)
		if err != nil {
			fmt.Printf("Cannot connect to %s \n", d)
			log.Println(err)

			return err
		}
		fmt.Printf("Connected to %s\n", destinations[i])
		defer func(connection quic.EarlyConnection, code quic.ApplicationErrorCode, s string) {
			err := connection.CloseWithError(code, s)
			if err != nil {

			}
		}(sessions[i], 0x1337, "Test finished!")

	}

	// ticker setup
	ticker := exponentialTicker(client.RequestsPerSec)
	done := make(chan bool)
	counter := make(chan int64, client.MaxConcurrentConnections)

	// https://yizhang82.dev/go-pattern-for-worker-queue
	// https://gobyexample.com/worker-pools

	client.setupTestDuration(done)
	util.SetupGracefulShutdown(func() {
		done <- true
	})

	// requests loop
external:
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
			err := client.newReq(sessions, id)
			if err != nil {
				return
			}
			<-counter
		}()

		select {
		case <-done:
			ticker.Stop()
			break external
		case <-ticker.C:
			continue external
		}
	}
	fmt.Printf("Waiting %d connections...\n", len(counter))
	wg.Wait()
	fmt.Println("Stopped")

	return nil
}

// https://pkg.go.dev/github.com/lucas-clemente/quic-go
// func newReq(session quic.EarlySession, id int64) error {
// crea e invia una nuova richiesta ai destinatari
func (client *Client) newReq(session []quic.EarlyConnection, id int64) error {

	// CREA LA RICHIESTA
	idReq := client.IdDevice + "_" + strconv.FormatInt(id, 10)
	req := util.RoPEMessage{ReqID: idReq,
		Log:  client.LoggerEnabled,
		Type: util.Request}

	dest := ForwardDecision(&req, client.Destinations)

	var idxdest = 0
	for i, d := range client.Destinations {
		if d == dest {
			idxdest = i
		}
	}

	stream, err := session[idxdest].OpenStream()
	if err != nil {
		return err
	}

	//util.LogEvent(idReq, util.Sent, "New Request", loggerEnabled, idDevice)

	// INVIA LA RICHIESTA
	encoder := gob.NewEncoder(stream)
	err = encoder.Encode(req)

	if err != nil {
		fmt.Printf("Error sending: %s\n", idReq)
		return err
	}

	// RICEVE LA RISPOSTA
	var packet util.RoPEMessage
	var rcv = false
	decoder := gob.NewDecoder(stream)
	var wg sync.WaitGroup
	var cnt int64 = 0
	for {
		cnt += 1
		err = decoder.Decode(&packet)
		if err == io.EOF || err != nil {
			break
		} else {
			rcv = true
		}
		wg.Add(1)
		go forwardResponse(packet, &wg)
		/*if packet.Type == util.Response {
			util.LogEvent(idReq, util.Received, "Response", loggerEnabled, idDevice)
		} else {
			util.LogEvent(idReq, util.ReceivedError, "Error", loggerEnabled, idDevice)
		}*/
	}
	wg.Wait()

	if !rcv {
		fmt.Printf("Timeout or broken: %s\n", idReq)
		//util.LogEvent(req.ReqID, util.Timeout, "Timeout response", loggerEnabled, idDevice)
		return err
	}

	// if resp.Type == util.Response {
	// 	util.LogEvent(idReq, util.Received, "Response", loggerEnabled, idDevice)
	// } else {
	// 	util.LogEvent(idReq, util.ReceivedError, "Error", loggerEnabled, idDevice)
	// }

	// fmt.Printf("Client: Received %s  %s\n", resp.ReqID, resp.Type)

	err = stream.Close()
	if err != nil {
		return err
	}

	return nil
}

// Gestisce la risposta ricevuta, asincrona
func forwardResponse(packet util.RoPEMessage, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("risposta ricevuta: type: ", packet.Type)
	fmt.Println("risposta ricevuta: ID: ", packet.ReqID)
	fmt.Println("risposta ricevuta: source: ", packet.Source)
	// Log traffic
	if ForwardSetLastResponse != nil {
		ForwardSetLastResponse(packet)
	}
}
