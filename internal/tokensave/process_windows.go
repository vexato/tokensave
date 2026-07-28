//go:build windows

package tokensave

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
)

func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}
func shellCommand(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, "cmd.exe", "/C", command)
}
