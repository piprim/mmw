package main

import (
	"log"
	"os"

	"github.com/piprim/mmw/cmd/mmw-cli/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
