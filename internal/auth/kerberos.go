package auth

import (
	"fmt"
	"os/exec"
	"strings"
)

type KerberosStatus struct {
	Authenticated bool
	Principal     string
}

func CheckKerberos() KerberosStatus {
	cmd := exec.Command("klist", "-s")
	if err := cmd.Run(); err != nil {
		return KerberosStatus{Authenticated: false}
	}

	// klist -s succeeded (exit 0) → ticket is valid, get principal from klist
	cmd = exec.Command("klist")
	out, err := cmd.Output()
	if err != nil {
		return KerberosStatus{Authenticated: true}
	}

	principal := parsePrincipal(string(out))
	return KerberosStatus{
		Authenticated: true,
		Principal:     principal,
	}
}

// Kinit runs kinit without a principal (uses system default) and pipes password via stdin.
func Kinit(password string) error {
	cmd := exec.Command("kinit")
	cmd.Stdin = strings.NewReader(password + "\n")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kinit failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func parsePrincipal(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Default principal:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Default principal:"))
		}
	}
	return ""
}
