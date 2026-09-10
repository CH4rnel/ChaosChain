// cmd/chaoschaind/main.go
package main

import (
	"fmt"
	"os"

	"github.com/CH4rnel/ChaosChain/app"
)

func main() {
	rootCmd := app.NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}