package client

import (
	"github.com/paolocastagno/go_rope/util"
)

/////////// Logic parsing //////////////

var ForwardDecision func(msg *util.RoPEMessage, destinations []string) string
var ForwardSetLastResponse func(util.RoPEMessage)

// ForwardingLogic definisce l'interfaccia per le logiche di inoltro
type ForwardingLogic interface {
	Init()
	Decision(*util.RoPEMessage, []string) string
	SetLastResponse(util.RoPEMessage)
}

/////////////// Helper functions ////////////

/*func LoadForwardingConf(confFile string) {
	config, err := toml.LoadFile(confFile)
	if err != nil {
		die("Error loading configuration", err.Error())
	} else {
		// retrieve data directly
		//logicName := config.Get("logic.name").(string)
		initFuncName := config.Get("logic.init").(string)

		fmt.Printf(initFuncName)

		// Itera su tutti i simboli registrati
		for _, sym := range runtime.Symbols() {
			// Verifica se il nome del simbolo corrisponde alla funzione cercata
			if strings.HasSuffix(sym.Name, funcName) {
				// Ottieni il valore della funzione dal simbolo
				funcValue := reflect.ValueOf(sym.Func)
				// Verifica se il valore ottenuto è una funzione
				if funcValue.Kind() == reflect.Func {
					return funcValue
				}
			}
		}
		// Se non viene trovata nessuna corrispondenza, restituisci un valore nullo
		return reflect.Value{}
	}
}*/

// Carica i plugin di logica di inoltro durante l'esecuzione del server
/*func LoadForwardingLogic(pluginPath string, config *toml.Tree) ForwardingLogic {
	p, err := plugin.Open(pluginPath)
	if err != nil {
		fmt.Println("Errore durante l'apertura del plugin:", err)
		os.Exit(1)
	}

	// Ottieni un puntatore all'oggetto ForwardingLogic dal plugin
	symForwardingLogic, err := p.Lookup("ForwardingLogic")
	if err != nil {
		fmt.Println("Simbolo ForwardingLogic non trovato nel plugin:", err)
		os.Exit(1)
	}

	// Converte il simbolo in un'interfaccia ForwardingLogic
	forwardingLogic, ok := symForwardingLogic.(ForwardingLogic)
	if !ok {
		fmt.Println("Il simbolo non è un'interfaccia ForwardingLogic")
		os.Exit(1)
	}

	// Inizializza la logica di inoltro del plugin
	forwardingLogic.Init(config)

	return forwardingLogic
}*/

/*func loadForwardingConf(confFile string, destinations string) {
	config, err := toml.LoadFile(confFile)
	if err != nil {
		die("Error loading configuration", err.Error())
	} else {
		// retrieve data directly
		logicName := config.Get("logic.name").(string)

		if isLogicSupported(logicName) {
			logicsMap[logicName](config) //chiama la logica corrispondente
		} else {
			die("No supported logic name specified")
		}
	}
}*/
