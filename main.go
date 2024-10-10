package main

import (
	"github.com/oxio/rediscp/cmd"
	"os"
)

func main() {
	err := cmd.NewCpCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}
