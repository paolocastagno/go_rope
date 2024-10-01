package util

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/go-ping/ping"
	"github.com/go-zeromq/zmq4"

	"gonum.org/v1/gonum/stat/distuv"
)

type RoPEMsgType string

const (
	Request        RoPEMsgType = "Request"
	Response       RoPEMsgType = "Response"
	QueueFull      RoPEMsgType = "QueueFull"
	ServerNotFound RoPEMsgType = "ServerNotFound"
	ServerTimeout  RoPEMsgType = "ServerTimeout"
	NoRoute        RoPEMsgType = "NoRoute"
	MessageLost    RoPEMsgType = "MessageLost" //Aggiunta per segnalare la perdita di pacchetti
)

type RoPEMessage struct {
	ReqID       string
	Type        RoPEMsgType
	Log         bool
	ResSize     int32
	Body        []byte
	Source      string
	Hop         string
	Destination string
}

// Setup a bare-bones TLS config for the server and the proxy
func GenerateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"RoPEProtocol"},
	}
}

/////////////////// PING ///////////////////////

func PingTest(pingAddr string, direction string, duration time.Duration, wgPing *sync.WaitGroup, c chan float64, idDevice string) {
	wgPing.Add(1)
	defer wgPing.Done()

	host := strings.Split(pingAddr, ":")

	fmt.Println("[Ping] Host:", host[0], direction)

	pinger, err := ping.NewPinger(host[0])
	if err != nil {
		fmt.Println("[Ping] Error:", err, direction)
		LogPing(PingError, 0*time.Millisecond, direction, err.Error(), idDevice)
		return
	}
	pinger.SetPrivileged(true)
	pinger.Count = -1
	pinger.Timeout = duration
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		fmt.Println("[Ping] Error:", err, direction)
		LogPing(PingError, 0*time.Millisecond, direction, err.Error(), idDevice)
		return
	}
	stats := pinger.Statistics()

	if stats.PacketLoss == 100 {
		fmt.Println("[Ping] FAILED", direction)
		LogPing(PingError, 0*time.Millisecond, direction, "All packet loss", idDevice)
	} else {
		fmt.Println("[Ping] SUCCESS", direction)
		fmt.Println("[Ping] Avg:", stats.AvgRtt, direction)
		LogPing(PingSuccess, stats.AvgRtt, direction, fmt.Sprintf("%+v\n", stats), idDevice)
	}
	fmt.Printf("%s: %+v\n", direction, stats)
	time.Sleep(5 * time.Second)
	c <- float64(stats.AvgRtt) / float64(time.Millisecond)
	// fmt.Printf("Sending %d back to proxy and finishing\n", int64((math.Ceil(float64(stats.AvgRtt) / float64(time.Millisecond)))))
	//s, _ := json.MarshalIndent(stats, "", "\t")
	//fmt.Println("[Ping] Res:", string(s))
}

// ///////////// MODEM /////////////////
func ZmqInfo(zmqAddr string, idDevice string) {

	//  Prepare our subscriber
	sub := zmq4.NewSub(context.Background())
	defer sub.Close()

	err := sub.Dial(zmqAddr)
	if err != nil {
		fmt.Printf("[ZMQ] Could not dial %v \n", err)
		return
	}

	err = sub.SetOption(zmq4.OptionSubscribe, "MONROE.META.DEVICE.MODEM")
	if err != nil {
		fmt.Printf("[ZMQ] Could not subscribe %v \n", err)
	}

	for i := 0; i < 1; i++ {
		// Read envelope
		msg, err := sub.Recv()
		if err != nil {
			fmt.Println("[ZMQ] Error: ", err)
		}
		fmt.Printf("%+v\n", msg)
		ExtractZmqInfo(msg, idDevice)
		//fmt.Printf("[%s] %s\n", msg.Frames[0], msg.Frames[1])
	}
}

type ZmqModem struct {
	InternalInterface string `json:"InternalInterface"`
	Operator          string `json:"Operator"`
	IPAddress         string `json:"IPAddress"`
	Frequency         uint   `json:"Frequency"`
	RSSI              int    `json:"RSSI"`
}

func ExtractZmqInfo(msg zmq4.Msg, idDevice string) {
	if len(msg.Frames) < 1 {
		fmt.Println("[ZMQ] Error: no Frame")
		return
	}

	data := string(msg.Frames[0])
	dataSplit := strings.SplitN(data, " ", 2)

	if len(dataSplit) < 2 {
		fmt.Println("[ZMQ] Error: no msgData")
		return
	}

	//topic := dataSplit[0]
	msgData := dataSplit[1]

	var modemInfo ZmqModem = ZmqModem{
		InternalInterface: "Unknown",
		Operator:          "Unknown",
		IPAddress:         "Unknown",
		Frequency:         0,
		RSSI:              0,
	}

	errj := json.Unmarshal([]byte(msgData), &modemInfo)
	if errj != nil {
		fmt.Println("error:", errj)
		return
	}

	fmt.Printf("%+v\n", modemInfo)

	LogMonroe(modemInfo, data, idDevice)

}

// //////////////     DELAY     //////////////////
func InitDelay(dist string, dist_params interface{}) interface{} {

	if dist_params == nil {
		panic("No delay or distribution specified")
	}
	fmt.Printf("Distribution (%s)\n", dist)
	var distribution interface{} = nil
	switch dist {
	case "uniform":
		params := dist_params.([]interface{})
		min, _ := time.ParseDuration(params[0].(string))
		max, _ := time.ParseDuration(params[1].(string))
		fmt.Printf("\t - interval (%d - %d)\n", min, max)
		distribution = &distuv.Uniform{
			Min: float64(int64(min) / int64(time.Second)),
			Max: float64(int64(max) / int64(time.Second)),
		}
	case "exponential":
		avg, _ := time.ParseDuration(dist_params.(string))
		rate := float64(time.Second) / float64(int64(avg))
		fmt.Printf("\t - rate %f\n", rate)
		distribution = distuv.Exponential{
			Rate: rate,
		}
	case "constant":
		distribution = dist_params
		fmt.Printf("\t - value %s\n", distribution)
	case "no delay":
		return nil
	default:
		panic("Distirbution " + dist + " not supported")
	}
	return distribution
}

func Delay(distribution interface{}, distType string) {
	var delay time.Duration
	switch distType {
	case "uniform":
		d := (distribution.(distuv.Uniform)).Rand()
		d, _ = math.Modf(float64(d) * float64(time.Second))
		delay = time.Duration(d * float64(time.Second))
		time.Sleep(delay)
	case "exponential":
		d := (distribution.(distuv.Exponential)).Rand()
		d, _ = math.Modf(float64(d) * float64(time.Second))
		delay = time.Duration(d * float64(time.Second))
		time.Sleep(delay)
	case "constant":
		delay, _ = time.ParseDuration(distribution.(string))
		time.Sleep(delay)
	case "no delay":
		return
	default:
		panic("Distirbution " + distType + " not supported")
	}

}

// ///////////// MOVING AVERAGE /////////////////
type Mavg struct {
	vec   []int64
	begin int64
	end   int64
	sum   int64
}

func NewMavg(size int64) Mavg {
	x := Mavg{}
	x.vec = make([]int64, 0, size)
	x.begin = 0
	x.end = 0
	x.sum = 0
	return x
}

func Mavg_push(x *Mavg, y int64) {
	// fmt.Printf("\tMavg_push(x.begin - %d, x.end - %d, x.sum - %f, len(x.vec) %d, y - %f)\n", x.begin, x.end, x.sum, len(x.vec), y)
	if len(x.vec) == 0 {
		x.vec = append(x.vec, y)
		x.sum = y
		x.end = x.end + 1
	} else {
		if len(x.vec) < cap(x.vec) {
			x.vec = append(x.vec, y)
			x.sum = x.sum + y
			x.end = x.end + 1
		} else {
			x.sum = x.sum - x.vec[x.begin] + y
			x.vec[x.begin] = y
			x.begin = int64(math.Mod(float64(x.begin+1), float64(len(x.vec))))
			x.end = int64(math.Mod(float64(x.end+1), float64(len(x.vec))))
		}
	}
}

func Mavg_eval(x Mavg, window int64) float64 {
	// fmt.Printf("\tMavg_eval(x.sum - %f, len(x.vec) %d): %f\n", x.sum, len(x.vec), float64(x.sum)/float64(window*int64(len(x.vec))))
	return float64(x.sum) / float64(window*int64(len(x.vec)))
}
