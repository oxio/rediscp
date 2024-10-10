package main

import (
	"os"
	"rediscp/cmd"
)

func main() {
	err := cmd.NewCpCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}
