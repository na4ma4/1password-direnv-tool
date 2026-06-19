//go:build windows

package protonpass

import "os/exec"

func findPassCLI() string {
	path, err := exec.LookPath("pass-cli")
	if err == nil {
		return path
	}

	return "C:\\Program Files\\ProtonPass\\pass-cli.exe"
}
