package main

import (
	"os"

	"github.com/dsuranov/canvasprobe/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
