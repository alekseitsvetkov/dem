package main

import (
	"os"

	"github.com/alekseitsvetkov/dem/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
