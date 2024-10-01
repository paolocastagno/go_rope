package config

import (
	"fmt"
	"github.com/pelletier/go-toml"
	"os"
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
		// retrieve data directly
		//config.Get("logic.file")

		initLogic(config) // Chiama direttamente la funzione di inizializzazione della logica di inoltro
	}
}

func Die(msg ...interface{}) {
	fmt.Println(msg...)
	os.Exit(1)
}

/*func supportedLogics() []string {
	logicsNames := make([]string, 0, len(logicsMap))
	for name := range logicsMap {
		logicsNames = append(logicsNames, name)
	}
	return logicsNames
}

func isLogicSupported(logicName string) bool {
	for _, name := range supportedLogics() {
		if name == logicName {
			return true
		}
	}
	return false
}*/

//reflection
/*func LoadForwardingConf(confFile string) {
	config, err := toml.LoadFile(confFile)
	if err != nil {
		Die("Error loading configuration", err.Error())
	} else {
		// retrieve data directly
		//logicName := config.Get("logic.name").(string)
		initFuncName := config.Get("logic.init").(string)

		// Ottieni il valore della funzione di inizializzazione dall'ambiente di runtime
		initFunc := reflect.ValueOf(initFuncName)
		if !initFunc.IsValid() {
			Die("Invalid init function:", initFuncName)
		}

		// Verifica se la funzione di inizializzazione è effettivamente una funzione
		if initFunc.Kind() != reflect.Func {
			Die("Init function is not a function:", initFuncName)
		}

		// Chiamata dinamica alla funzione di inizializzazione senza argomenti
		initFunc.Call([]reflect.Value{})
	}
}*/
