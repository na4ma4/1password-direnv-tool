//go:build darwin

package protonpass

import "os/exec"

func findPassCLI() string {
	path, err := exec.LookPath("pass-cli")
	if err == nil {
		return path
	}

	return "/opt/homebrew/opt/proton-pass-cli/bin/pass-cli"
}
