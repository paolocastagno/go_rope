package server

import (
	quic "github.com/quic-go/quic-go"

	"github.com/paolocastagno/go_rope/util"
)

/////////// Logic parsing //////////////

var ForwardDecision func(*util.RoPEMessage, *map[string]quic.EarlyConnection, int64) bool
var ForwardBlock func(*util.RoPEMessage, quic.Stream)
var ForwardSetLastResponse func(*util.RoPEMessage)

/*var logicsMap = map[string]func(*toml.Tree){
	"reply": parseReply,
	//"replygame":  parseGame,
	//"replyrelay": parseRReply,
}

func parseReply(config *toml.Tree) {

	dest := config.Get("variables.destination")
	// rtt := config.Get("variables.rtt")
	wt := config.Get("variables.workTime")
	pkts := config.Get("variables.packets")

	// InitReply(wt, dest, rtt, pkts)
	InitReply(wt, dest, pkts)
	ForwardDecision = ReplyDecision
	ForwardBlock = nil
	ForwardSetLastResponse = ReplySetLastResponse
}*/

/*func parseGame(config *toml.Tree) {

	dest := config.Get("variables.destination")
	wt := config.Get("variables.workTime")

	InitReplyGame(wt, dest)
	ForwardDecision = ReplyGameDecision
	ForwardBlock = nil
	ForwardSetLastResponse = ReplySetLastResponse
}*/

/*func parseRReply(config *toml.Tree) { //

	dest := config.Get("variables.destination")
	// rtt := config.Get("variables.rtt")
	wt := config.Get("variables.workTime")
	pkts := config.Get("variables.packets")

	// InitRReply(wt, dest, rtt, pkts)
	InitRReply(wt, dest, pkts)
	ForwardDecision = RReplyDecision
	ForwardBlock = RReplyBlock
	ForwardSetLastResponse = RReplySetLastResponse
}*/

/////////////// Helper functions ////////////

/*func loadForwardingConf(confFile string) { //, rtt []float64) { addr []string
	config, err := toml.LoadFile(confFile)
	if err != nil {
		log.Fatal(err)
	} else {
		// Add to the configuration the available connections, if any
		//if len(addr) != 0 {
		//	fmt.Println("Adding destinations to appconfig: ", addr)
		//	config.Set("variables.destination", addr)
		}
		// 	fmt.Println("Adding rtt to appconfig: ", rtt)
		// 	config.Set("variables.destination", rtt)
		// }
		logicName := config.Get("logic.name").(string)
		if isLogicSupported(logicName) {
			logicsMap[logicName](config)
		} else {
			die("Application is not supported (" + logicName + ")")
		}
	}
}*/
