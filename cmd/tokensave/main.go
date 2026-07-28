package main

import (
	"fmt"
	"os"

	"github.com/vexato/tokensave/internal/tokensave"
)

func main() {
	if err := tokensave.Execute(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if code, ok := tokensave.ChildExitCode(err); ok {
			os.Exit(code)
		}
		fmt.Fprintln(os.Stderr, "tokensave:", err)
		os.Exit(2)
	}
}
