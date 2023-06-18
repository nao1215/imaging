//go:build linux

package main

import (
	"os/exec"
)

func openBrowser(targetURL string) bool {
	return exec.Command("xdg-open", targetURL).Start() == nil
}
