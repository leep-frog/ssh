package main

import (
	"github.com/leep-frog/command/sourcerer"
	"github.com/leep-frog/ssh"
)

func main() {
	sourcerer.Source([]sourcerer.CLI{ssh.CLI()})
}
