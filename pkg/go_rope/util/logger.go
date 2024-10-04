package util

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxdb2API "github.com/influxdata/influxdb-client-go/v2/api"
)

var client influxdb2.Client
var writeAPI influxdb2API.WriteAPI

// DEFAULT VARIABLES
var DefConfLog = LoggerConf{
	LoggerToken:         "",
	LoggerAddress:       "localhost:8086",
	LoggerBucket:        "rope",
	LoggerOrg:           "unito",
	LoggerBatchSize:     5000,
	LoggerFlushInterval: 5000,
}

// GLOBAL VARIABLES
var bucket string
var org string
var loggerToken string
var loggerAddress string
var loggerBatchSize uint
var loggerFlushInterval uint

var LoggerCfg LoggerConf

type RoPEEventType string

const (
	Sent          RoPEEventType = "sent"
	Received      RoPEEventType = "received"
	ReceivedError RoPEEventType = "receivedError"
	FullQueue     RoPEEventType = "fullQueue"
	Enqueued      RoPEEventType = "enqueued"
	Timeout       RoPEEventType = "timeout"
	Dequeued      RoPEEventType = "dequeued"
	Processed     RoPEEventType = "processed"
	PingAverage   RoPEEventType = "ping"
	RoutingErr    RoPEEventType = "NoRoute"
)

const retryConnections = 40
const waitRetry = 2000

type LoggerConf struct {
	LoggerToken         string `json:"loggerToken"`
	LoggerAddress       string `json:"loggerAddress"`
	LoggerBucket        string `json:"loggerBucket"`
	LoggerOrg           string `json:"loggerOrg"`
	LoggerBatchSize     uint   `json:"loggerBatchSize"`
	LoggerFlushInterval uint   `json:"loggerFlushInterval"`
}

func IsLoggerEnabled() bool {
	return writeAPI != nil
}

func SetLoggerParam() {
	flag.StringVar(&loggerToken, "loggerToken", DefConfLog.LoggerToken, "influxdb token")
	flag.StringVar(&loggerAddress, "loggerAddress", DefConfLog.LoggerAddress, "influxdb address")
	flag.StringVar(&bucket, "loggerBucket", DefConfLog.LoggerBucket, "influxdb bucket")
	flag.StringVar(&org, "loggerOrg", DefConfLog.LoggerOrg, "influxdb org")
	flag.UintVar(&loggerBatchSize, "loggerBatchSize", DefConfLog.LoggerBatchSize, "influxdb batch size to reach to flush")
	flag.UintVar(&loggerFlushInterval, "loggerFlushInterval", DefConfLog.LoggerFlushInterval, "influxdb milliseconds interval to force flush")
}

func SetLoggerParamFromConf(conf LoggerConf) {
	loggerToken = conf.LoggerToken
	loggerAddress = conf.LoggerAddress
	bucket = conf.LoggerBucket
	org = conf.LoggerOrg
	loggerBatchSize = conf.LoggerBatchSize
	loggerFlushInterval = conf.LoggerFlushInterval
	// Write the logger configuration to a global variable accessibe from outside the module (substitue with a function)
	LoggerCfg = conf
}

func InitLogger() error {
	if loggerAddress == "" || loggerToken == "" {
		fmt.Println("logger parameters empty: logger not in use")
		return nil
	}
	fmt.Println(loggerToken)

	// You can generate a Token from the "Tokens Tab" in the UI
	//const token = "x-VLGmrax87Va_rQRfStrkD2tMWFjPDrV5NzxBomiZFkHm4Ax1k-ur6i79zSd5-b3I9aFTl6ZC8Ng3phqJ_Dzg=="

	client := influxdb2.NewClientWithOptions("http://"+loggerAddress, loggerToken, influxdb2.DefaultOptions().SetBatchSize(loggerBatchSize).SetFlushInterval(loggerFlushInterval).SetPrecision(time.Nanosecond))

	var err error
	for i := 0; i < retryConnections; i++ {
		_, err = client.Ready(context.Background())
		if err == nil {
			break
		}
		fmt.Printf("influxdb NOT ready... wait %d ms\n", waitRetry)
		time.Sleep(waitRetry * time.Millisecond)
	}

	if err != nil {
		fmt.Printf("influxdb host logger NOT ready: %s\n", loggerAddress)
		return err
	} else {
		fmt.Printf("influxdb host logger host ready: %s\n", loggerAddress)
	}

	writeAPI = client.WriteAPI(org, bucket)

	if writeAPI == nil {
		return errors.New("writeAPI creation error")
	}

	return nil
}

func SetupGracefulShutdown(shutdownFunc func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Println("got interrupt signal: ", <-sigChan)
		shutdownFunc()
	}()
}

func CloseLogger() {
	if writeAPI != nil {
		fmt.Println("Flushing logger...")
		writeAPI.Flush()
		writeAPI = nil
	}

	if client != nil {
		client.Close()
		client = nil
	}
}

func LogEvent(idRequest string, eventType RoPEEventType, comment string, log bool, idDevice string) {
	if IsLoggerEnabled() && log {
		p := influxdb2.NewPoint(
			"packet",
			map[string]string{
				// Global
				"idDevice":  idDevice,
				"idRequest": idRequest,
				"eventType": string(eventType),
				"timestamp": strconv.FormatInt(time.Now().UnixNano(), 10),
			},
			map[string]interface{}{
				"comment": comment,
			},
			time.Now(),
		)
		// write asynchronously
		writeAPI.WritePoint(p)
	}
}

/////////////////////////////

type PingEventType string

const (
	PingSuccess PingEventType = "success"
	PingError   PingEventType = "error"
)

func LogPing(event PingEventType, rtt time.Duration, dir string, comment string, idDevice string) {
	if IsLoggerEnabled() {
		p := influxdb2.NewPoint(
			"ping",
			map[string]string{
				// Global
				"idDevice":  idDevice,
				"eventType": string(event),
				"rtt":       fmt.Sprint(rtt),
				"direction": string(dir),
			},
			map[string]interface{}{
				"comment": comment,
			},
			time.Now(),
		)
		// write asynchronously
		writeAPI.WritePoint(p)
	}
}

func LogMonroe(modemInfo ZmqModem, comment string, idDevice string) {
	if IsLoggerEnabled() {
		p := influxdb2.NewPoint(
			"monroe",
			map[string]string{
				// Global
				"idDevice":  idDevice,
				"operator":  modemInfo.Operator,
				"interface": modemInfo.InternalInterface,
				"ipaddr":    modemInfo.IPAddress,
				"frequency": fmt.Sprint(modemInfo.Frequency),
				"RSSI":      fmt.Sprint(modemInfo.RSSI),
			},
			map[string]interface{}{
				"comment": comment,
			},
			time.Now(),
		)
		// write asynchronously
		writeAPI.WritePoint(p)
	}
}

/*
Msg{Frames:{"MONROE.META.DEVICE.MODEM.8934041918060329972.SIGNAL {	\"SequenceNumber\":1340,
																	\"Timestamp\":1615501577.1222651,
																	\"DataVersion\":3,
																	\"DataId\":\"MONROE.META.DEVICE.MODEM\",
																	\"InternalInterface\":\"op0\",
																	\"ICCID\":\"8934041918060329972\",
																	\"IMSI\":\"214042900938457\",
																	\"IMEI\":\"359072060474182\",
																	\"Operator\":\"Yoigo Internet\",
																	\"IPAddress\":\"100.77.30.123\",
																	\"InterfaceName\":\"nlw_1\",
																	\"IMSIMCCMNC\":21404,
																	\"NWMCCMNC\":21404,
																	\"LAC\":65535,
																	\"CID\":72209510,
																	\"RSRP\":-94,
																	\"Frequency\":1800,
																	\"RSSI\":-64,
																	\"RSRQ\":-10,
																	\"DeviceMode\":5,
																	\"DeviceSubmode\":0,
																	\"Band\":3,
																	\"DeviceState\":3,
																	\"PCI\":65535}"}}
*/
