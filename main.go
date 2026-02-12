// main.go

package main

import (
	"log"
	"os"
)

// print help message
func printHelp(cmd string) {
	log.Printf(`> Usage:

  * fetch, summarize, cache, and serve RSS feed items
    %[1]s [CONFIG_FILEPATH]
`, cmd)
}

func main() {
	cmd := os.Args[0]
	args := os.Args[1:]

	if len(args) > 0 {
		configFilepath := args[0]
		if conf, err := readConfig(configFilepath); err == nil {
			run(conf)

			os.Exit(0)
		} else {
			log.Printf("> failed to read config: %s", err)
		}
	} else {
		printHelp(cmd)
	}

	os.Exit(1)
}
