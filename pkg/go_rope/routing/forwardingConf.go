package routing

import (
	"fmt"
	"github.com/paolocastagno/go_rope/util"
	"os"
)

/////////// Logic parsing //////////////

var ForwardDecision func(*util.RoPEMessage)
var ForwardSetLastResponse func(*util.RoPEMessage)

/*var logicsMap = map[string]func(*toml.Tree){
	//"probability":        parseProbability,
	//"probabilityLatency": parsePL,
	"fixed": parseFixed,
}*/

/*func parseProbability(config *toml.Tree) {
	p := config.Get("variables.prob")
	d := config.Get("variables.dest")

	InitWeightedRandom(p, d)
	ForwardDecision = WeightedRandomDecision
	ForwardSetLastResponse = WeightedRandomSetLastResponse
}

func parsePL(config *toml.Tree) {
	p := config.Get("variables.prob")
	d := config.Get("variables.dest")

	lcu := config.Get("latency.up_cfg")
	ltu := config.Get("latency.up_type")

	lcd := config.Get("latency.down_cfg")
	ltd := config.Get("latency.down_type")

	InitPL(p, d, lcu, ltu, lcd, ltd)
	ForwardDecision = PLDecision
	ForwardSetLastResponse = PLSetLastResponse
}*/

/*func parseFixed(config *toml.Tree) {
	d := config.Get("variables.dest")
	InitFixed(d)
	ForwardDecision = FixedDecision
	ForwardSetLastResponse = FixedSetLastResponse
}*/

/////////////// Helper functions ////////////

func die(msg ...interface{}) {
	fmt.Println(msg...)
	os.Exit(1)
}

/*
func loadForwardingConf(confFile string, rtt map[string]float64) {
	config, err := toml.LoadFile(confFile)
	if err != nil {
		log.Fatal(err)
	}
	// else {
	// 	config.Set("variables.rtt", rtt)
	// 	fmt.Printf("rtt: %f\n", rtt)
	// }

	if err != nil {
		die("Error loading configuration", err.Error())
	} else {
		// retrieve data directly
		logicName := config.Get("logic.name").(string)

		if isLogicSupported(logicName) {
			logicsMap[logicName](config)
		} else {
			die("No supported logic name specified (" + logicName + ")")
		}
	}
}*/
