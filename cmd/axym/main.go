package main

import (
	"os"

	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
)

var version = "dev"

func init() { actioncontract.SetConsumerVersion(version) }

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr))
}
