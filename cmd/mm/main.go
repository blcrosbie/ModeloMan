package main

import (
	"log"
	"os"

	"github.com/bcrosbie/modeloman/internal/mm/cli"
)

func main() {
	if err := cli.Run(os.Args[1:], "mm"); err != nil {
		log.Fatalf("%v", err)
	}
}
