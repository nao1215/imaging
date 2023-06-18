//go:build !darwin && !linux && !windows

package main

func openBrowser(string) bool {
	return false
}
