package readonly

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestRestrictedShell_CommandChaining tests that the restricted shell properly
// blocks command chaining attempts using various shell metacharacters.
func TestRestrictedShell_CommandChaining(t *testing.T) {
	// Create a temporary shell script file for testing
	tmpfile, err := os.CreateTemp("", "fluid-readonly-shell-test-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Write the restricted shell script
	if _, err := tmpfile.Write([]byte(RestrictedShellScript)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Make it executable
	if err := os.Chmod(tmpfile.Name(), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		command     string
		shouldBlock bool
		description string
	}{
		// Command chaining attempts that should be blocked
		{
			name:        "semicolon_chaining",
			command:     "cat /etc/hosts; rm -rf /",
			shouldBlock: true,
			description: "semicolon command chaining should be blocked",
		},
		{
			name:        "double_ampersand_chaining",
			command:     "ls /etc && rm -rf /",
			shouldBlock: true,
			description: "&& command chaining should be blocked",
		},
		{
			name:        "double_pipe_chaining",
			command:     "false || rm -rf /",
			shouldBlock: true,
			description: "|| command chaining should be blocked",
		},
		{
			name:        "command_substitution_dollar_paren",
			command:     "echo $(rm -rf /)",
			shouldBlock: true,
			description: "$() command substitution should be blocked",
		},
		{
			name:        "command_substitution_backticks",
			command:     "echo `rm -rf /`",
			shouldBlock: true,
			description: "backtick command substitution should be blocked",
		},
		{
			name:        "process_substitution_input",
			command:     "cat <(rm -rf /)",
			shouldBlock: true,
			description: "<() process substitution should be blocked",
		},
		{
			name:        "process_substitution_output",
			command:     "echo hello >(rm -rf /)",
			shouldBlock: true,
			description: ">() process substitution should be blocked",
		},
		// Valid commands that should be allowed
		{
			name:        "simple_cat",
			command:     "cat /etc/hosts",
			shouldBlock: false,
			description: "simple cat command should be allowed",
		},
		{
			name:        "pipe_to_grep",
			command:     "ps aux | grep nginx",
			shouldBlock: false,
			description: "pipe to grep should be allowed",
		},
		{
			name:        "multiple_pipes",
			command:     "cat /etc/hosts | sort | uniq",
			shouldBlock: false,
			description: "multiple pipes should be allowed",
		},
		// Edge cases
		{
			name:        "quoted_semicolon",
			command:     "echo 'hello; world'",
			shouldBlock: false,
			description: "semicolon in quotes should be allowed",
		},
		{
			name:        "semicolon_then_destructive",
			command:     "echo hello; sudo rm -rf /",
			shouldBlock: true,
			description: "semicolon followed by sudo should be blocked",
		},
		{
			name:        "and_then_destructive",
			command:     "true && chmod 777 /etc/passwd",
			shouldBlock: true,
			description: "&& followed by chmod should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute the shell with SSH_ORIGINAL_COMMAND set
			cmd := exec.Command(tmpfile.Name())
			cmd.Env = append(os.Environ(), "SSH_ORIGINAL_COMMAND="+tt.command)

			output, err := cmd.CombinedOutput()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					t.Fatalf("unexpected error running shell: %v", err)
				}
			}

			if tt.shouldBlock {
				// Command should be blocked (exit code 126 or 1)
				if exitCode == 0 {
					t.Errorf("%s: expected command to be blocked but it succeeded\nCommand: %s\nOutput: %s",
						tt.description, tt.command, output)
				} else if !strings.Contains(string(output), "ERROR:") {
					t.Errorf("%s: command blocked but no error message shown\nCommand: %s\nOutput: %s",
						tt.description, tt.command, output)
				}
			} else {
				// Command should succeed
				if exitCode != 0 && exitCode != 1 {
					// Exit code 1 is acceptable for commands that legitimately fail
					// (e.g., grep with no matches), but 126 means blocked by shell
					if exitCode == 126 {
						t.Errorf("%s: expected command to succeed but it was blocked\nCommand: %s\nOutput: %s",
							tt.description, tt.command, output)
					}
				}
			}
		})
	}
}

// TestRestrictedShell_InteractiveLoginBlocked tests that interactive login
// (without SSH_ORIGINAL_COMMAND) is denied.
func TestRestrictedShell_InteractiveLoginBlocked(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "fluid-readonly-shell-test-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(RestrictedShellScript)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(tmpfile.Name(), 0755); err != nil {
		t.Fatal(err)
	}

	// Execute without SSH_ORIGINAL_COMMAND
	cmd := exec.Command(tmpfile.Name())
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("expected interactive login to be denied")
	}

	if !strings.Contains(string(output), "Interactive login is not permitted") {
		t.Errorf("expected interactive login error message, got: %s", output)
	}
}

// TestRestrictedShell_OutputRedirectionBlocked tests that output redirection
// is properly blocked.
func TestRestrictedShell_OutputRedirectionBlocked(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "fluid-readonly-shell-test-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(RestrictedShellScript)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(tmpfile.Name(), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		command string
	}{
		{"single_redirect", "echo hello > /tmp/out"},
		{"double_redirect", "echo hello >> /tmp/out"},
		{"redirect_with_pipe", "cat /etc/hosts | sort > /tmp/out"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(tmpfile.Name())
			cmd.Env = append(os.Environ(), "SSH_ORIGINAL_COMMAND="+tt.command)

			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("expected command %q to be blocked", tt.command)
			}

			if !strings.Contains(string(output), "redirection is not permitted") {
				t.Errorf("expected redirection error message for %q, got: %s", tt.command, output)
			}
		})
	}
}
