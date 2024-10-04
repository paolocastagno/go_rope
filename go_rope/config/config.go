package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

// ForwardingConfig rappresenta la configurazione comune per l'inoltro
type ForwardingConfig struct {
	LogicName    string
	Destination  string
	RequestSize  int64
	ResponseSize int64
	// Aggiungi altri campi comuni necessari per la configurazione di inoltro
	// ...
}

// LoadForwardingConf carica la configurazione di inoltro da un file TOML specificato
func LoadForwardingConf(confFile string, initLogic func(*toml.Tree)) {
	config, err := toml.LoadFile(confFile)

	if err != nil {
		Die("Error loading configuration", err.Error())
	} else {
		initLogic(config)
	}
}

func Die(msg ...interface{}) {
	fmt.Println(msg...)
	os.Exit(1)
}
