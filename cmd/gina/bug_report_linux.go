//go:build linux

package main

import (
	"context"
	"os/exec"
)

// openBrowser hands targetURL to the desktop's URL handler. The URL is built in
// bug_report.go from a constant github.com address and a query-escaped body, and
// it is passed as an argument rather than through a shell, so nothing in it can
// become a command.
func openBrowser(ctx context.Context, targetURL string) bool {
	return exec.CommandContext(ctx, "xdg-open", targetURL).Start() == nil //nolint:gosec // see above
}
