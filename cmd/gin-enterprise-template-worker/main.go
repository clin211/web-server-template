package main

import (
	"fmt"
	"github.com/clin211/gin-enterprise-template/cmd/gin-enterprise-template-apiserver/app"
	"os"
)

func main() {
	if err := app.NewBackgroundCommand("worker").Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
