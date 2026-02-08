package readonly

import (
	"context"
	"fmt"
	"strings"
)

// SSHRunFunc executes a command on a remote host via SSH.
// Returns stdout, stderr, exit code, and error.
type SSHRunFunc func(ctx context.Context, command string) (stdout, stderr string, exitCode int, err error)

// PrepareResult contains the outcome of preparing a golden VM for read-only access.
type PrepareResult struct {
	UserCreated       bool
	ShellInstalled    bool
	CAKeyInstalled    bool
	SSHDConfigured    bool
	PrincipalsCreated bool
	SSHDRestarted     bool
}

// Prepare configures a golden VM for read-only access via the fluid-readonly user.
// All steps are idempotent. The sshRun function is used to execute commands on the VM.
//
// Steps:
//  1. Create fluid-readonly user with restricted shell
//  2. Install restricted shell script
//  3. Copy CA pub key for certificate verification
//  4. Configure sshd to trust the CA key
//  5. Set up authorized principals for fluid-readonly
//  6. Restart sshd
func Prepare(ctx context.Context, sshRun SSHRunFunc, caPubKey string) (*PrepareResult, error) {
	if strings.TrimSpace(caPubKey) == "" {
		return nil, fmt.Errorf("CA public key is required")
	}

	result := &PrepareResult{}

	// 1. Install restricted shell script at /usr/local/bin/fluid-readonly-shell
	shellCmd := fmt.Sprintf("cat > /usr/local/bin/fluid-readonly-shell << 'FLUID_SHELL_EOF'\n%sFLUID_SHELL_EOF\nchmod 755 /usr/local/bin/fluid-readonly-shell", RestrictedShellScript)
	stdout, stderr, code, err := sshRun(ctx, shellCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("install restricted shell: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	result.ShellInstalled = true

	// 2. Create fluid-readonly user (idempotent - ignore if exists)
	userCmd := `id fluid-readonly >/dev/null 2>&1 || useradd -r -s /usr/local/bin/fluid-readonly-shell -d /nonexistent -M fluid-readonly`
	stdout, stderr, code, err = sshRun(ctx, userCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("create fluid-readonly user: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	// Ensure the shell is correct even if user already existed
	_, _, _, _ = sshRun(ctx, "usermod -s /usr/local/bin/fluid-readonly-shell fluid-readonly")
	result.UserCreated = true

	// 3. Copy CA pub key to /etc/ssh/fluid_ca.pub
	caCmd := fmt.Sprintf("cat > /etc/ssh/fluid_ca.pub << 'FLUID_CA_EOF'\n%s\nFLUID_CA_EOF\nchmod 644 /etc/ssh/fluid_ca.pub", strings.TrimSpace(caPubKey))
	stdout, stderr, code, err = sshRun(ctx, caCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("install CA pub key: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	result.CAKeyInstalled = true

	// 4. Configure sshd to trust the CA key (idempotent)
	sshdCmds := []string{
		// Add TrustedUserCAKeys if not present
		`grep -q 'TrustedUserCAKeys /etc/ssh/fluid_ca.pub' /etc/ssh/sshd_config || echo 'TrustedUserCAKeys /etc/ssh/fluid_ca.pub' >> /etc/ssh/sshd_config`,
		// Add AuthorizedPrincipalsFile if not present
		`grep -q 'AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u' /etc/ssh/sshd_config || echo 'AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u' >> /etc/ssh/sshd_config`,
	}
	for _, cmd := range sshdCmds {
		stdout, stderr, code, err = sshRun(ctx, cmd)
		if err != nil || code != 0 {
			return result, fmt.Errorf("configure sshd: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
		}
	}
	result.SSHDConfigured = true

	// 5. Create authorized_principals directory and file for fluid-readonly
	principalsCmds := []string{
		"mkdir -p /etc/ssh/authorized_principals",
		"echo 'fluid-readonly' > /etc/ssh/authorized_principals/fluid-readonly",
		"chmod 644 /etc/ssh/authorized_principals/fluid-readonly",
	}
	for _, cmd := range principalsCmds {
		stdout, stderr, code, err = sshRun(ctx, cmd)
		if err != nil || code != 0 {
			return result, fmt.Errorf("create principals: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
		}
	}
	result.PrincipalsCreated = true

	// 6. Restart sshd to apply changes
	// Try systemctl first, fall back to service command
	restartCmd := `systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || service sshd restart 2>/dev/null || service ssh restart`
	stdout, stderr, code, err = sshRun(ctx, restartCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("restart sshd: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	result.SSHDRestarted = true

	return result, nil
}
