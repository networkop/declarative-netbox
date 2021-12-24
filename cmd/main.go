package main

import (
	"fmt"
	"os"

	"github.com/networkop/declarative-netbox/cmd/cmd"
)

var (
	Version   string
	GitCommit string
)

func main() {

	if err := cmd.Execute(Version, GitCommit); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
