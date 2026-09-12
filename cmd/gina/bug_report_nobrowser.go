//go:build !darwin && !linux && !windows

package main

import "context"

// openBrowser reports that this system has no URL handler gina knows how to
// call, so bug-report prints the template instead.
func openBrowser(context.Context, string) bool {
	return false
}
