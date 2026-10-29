//go:build !unix

package main

import "os/exec"

// Releases target Linux and macOS. Other platforms retain CommandContext's
// direct-child cancellation and the same bounded pipe wait.
func configureCommand(cmd *exec.Cmd) func() {
	return func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
}
