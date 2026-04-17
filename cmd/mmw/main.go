package main

import (
	"log"
	"os"

	"github.com/piprim/mmw/cmd/mmw/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
