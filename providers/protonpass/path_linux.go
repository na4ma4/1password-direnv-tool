//go:build linux

package protonpass

import "os/exec"

func findPassCLI() string {
	path, err := exec.LookPath("pass-cli")
	if err == nil {
		return path
	}

	return "/usr/local/bin/pass-cli"
}
