package main

import (
	"fmt"
	"os"

	"github.com/ngerakines/auroraops/cmd/auroraops/internal"
)

func main() {
	if err := internal.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
