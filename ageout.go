// Command ageout deletes files whose last modified time (or, with
// -created, creation time) is older than a configured age threshold.
package main

import (
	"os"

	"github.com/sig9org/ageout/internal/cli"
	"github.com/sig9org/ageout/internal/logx"
)

func main() {
	if err := cli.Main(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		logx.New(os.Stderr, false).Errorf("ageout: %v", err)
		os.Exit(1)
	}
}
