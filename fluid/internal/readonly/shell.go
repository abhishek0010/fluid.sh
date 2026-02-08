package readonly

// RestrictedShellScript is the server-side restricted shell installed at
// /usr/local/bin/fluid-readonly-shell on golden VMs. It blocks destructive
// commands as a defense-in-depth layer behind the client-side allowlist.
const RestrictedShellScript = `#!/bin/bash
# fluid-readonly-shell - restricted shell for read-only VM access.
# Installed by: fluid source prepare
# This shell is set as the login shell for the fluid-readonly user.
# It only allows commands passed via SSH_ORIGINAL_COMMAND (no interactive login).

set -euo pipefail

# Deny interactive login - require SSH_ORIGINAL_COMMAND
if [ -z "${SSH_ORIGINAL_COMMAND:-}" ]; then
    echo "ERROR: Interactive login is not permitted. This account is for read-only SSH commands only." >&2
    exit 1
fi

CMD="$SSH_ORIGINAL_COMMAND"

# Blocked command patterns (destructive operations)
BLOCKED_PATTERNS=(
    "^sudo "
    "^su "
    "^rm "
    "^mv "
    "^cp "
    "^dd "
    "^kill "
    "^killall "
    "^pkill "
    "^shutdown "
    "^reboot "
    "^halt "
    "^poweroff "
    "^init "
    "^telinit "
    "^chmod "
    "^chown "
    "^chgrp "
    "^useradd "
    "^userdel "
    "^usermod "
    "^groupadd "
    "^groupdel "
    "^groupmod "
    "^passwd "
    "^chpasswd "
    "^mkfs"
    "^mount "
    "^umount "
    "^fdisk "
    "^parted "
    "^lvm "
    "^mdadm "
    "^wget "
    "^curl "
    "^scp "
    "^rsync "
    "^ftp "
    "^sftp "
    "^python"
    "^perl "
    "^ruby "
    "^node "
    "^bash "
    "^sh "
    "^zsh "
    "^dash "
    "^csh "
    "^vi "
    "^vim "
    "^nano "
    "^emacs "
    "^sed -i"
    "^tee "
    "^install "
    "^make "
    "^gcc "
    "^g++ "
    "^cc "
    "^iptables "
    "^ip6tables "
    "^nft "
    "^systemctl start"
    "^systemctl stop"
    "^systemctl restart"
    "^systemctl reload"
    "^systemctl enable"
    "^systemctl disable"
    "^systemctl daemon"
    "^systemctl mask"
    "^systemctl unmask"
    "^systemctl edit"
    "^systemctl set"
    "^apt install"
    "^apt remove"
    "^apt purge"
    "^apt autoremove"
    "^apt-get "
    "^dpkg -i"
    "^dpkg --install"
    "^dpkg --remove"
    "^dpkg --purge"
    "^rpm -i"
    "^rpm --install"
    "^rpm -e"
    "^rpm --erase"
    "^yum "
    "^dnf "
    "^pip install"
    "^pip uninstall"
    "^pip3 install"
    "^pip3 uninstall"
)

# Check each pipe segment
IFS='|' read -ra SEGMENTS <<< "$CMD"
for segment in "${SEGMENTS[@]}"; do
    # Trim leading whitespace
    segment="${segment#"${segment%%[![:space:]]*}"}"

    for pattern in "${BLOCKED_PATTERNS[@]}"; do
        if echo "$segment" | grep -qE "$pattern"; then
            echo "ERROR: Command blocked by restricted shell: $segment" >&2
            exit 126
        fi
    done
done

# Block output redirection
if echo "$CMD" | grep -qE '[^"'"'"']>[^&]|[^"'"'"']>>'; then
    echo "ERROR: Output redirection is not permitted." >&2
    exit 126
fi

# Execute the command
exec /bin/bash -c "$CMD"
`
