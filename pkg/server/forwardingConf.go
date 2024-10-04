package server

import (
	"github.com/quic-go/quic-go"

	"github.com/paolocastagno/go_rope/pkg/util"
)

/////////// Logic parsing //////////////

var ForwardDecision func(*util.RoPEMessage, *map[string]quic.EarlyConnection, int64) bool
var ForwardBlock func(*util.RoPEMessage, quic.Stream)
var ForwardSetLastResponse func(*util.RoPEMessage)
var ForwardSetLastResponseBlock func(*util.RoPEMessage, quic.Stream)
