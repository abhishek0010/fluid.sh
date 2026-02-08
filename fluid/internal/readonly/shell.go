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

# Block command substitution and subshells
# Check for $(...), backticks, <(...), >(...)
if echo "$CMD" | grep -qE '\$\(|` + "`" + `|<\(|>\('; then
    echo "ERROR: Command substitution and subshells are not permitted." >&2
    exit 126
fi

# Block output redirection
if echo "$CMD" | grep -qE '[^"'"'"']>[^&]|[^"'"'"']>>'; then
    echo "ERROR: Output redirection is not permitted." >&2
    exit 126
fi

# Block newlines (commands must be single-line)
if [[ "$CMD" == *$'\n'* ]]; then
    echo "ERROR: Multi-line commands are not permitted." >&2
    exit 126
fi

# Split command on all shell separators: | || ; && (and newlines, already blocked above)
# We need to parse the command to split on these operators outside of quotes.
# For defense-in-depth, we'll use a bash function to split properly.

# Parse and validate each segment
parse_and_validate_segments() {
    local cmd="$1"
    local segment=""
    local in_single_quote=false
    local in_double_quote=false
    local prev_char=""
    local i
    
    for (( i=0; i<${#cmd}; i++ )); do
        local char="${cmd:$i:1}"
        local next_char="${cmd:$((i+1)):1}"
        
        # Track quote state
        if [[ "$char" == "'" && "$in_double_quote" == false && "$prev_char" != "\\" ]]; then
            in_single_quote=$( [[ "$in_single_quote" == true ]] && echo false || echo true )
            segment+="$char"
        elif [[ "$char" == '"' && "$in_single_quote" == false && "$prev_char" != "\\" ]]; then
            in_double_quote=$( [[ "$in_double_quote" == true ]] && echo false || echo true )
            segment+="$char"
        # Check for separators outside quotes
        elif [[ "$in_single_quote" == false && "$in_double_quote" == false ]]; then
            if [[ "$char" == "|" ]]; then
                # Check for ||
                if [[ "$next_char" == "|" ]]; then
                    validate_segment "$segment"
                    segment=""
                    ((i++))  # Skip next |
                else
                    validate_segment "$segment"
                    segment=""
                fi
            elif [[ "$char" == ";" ]]; then
                validate_segment "$segment"
                segment=""
            elif [[ "$char" == "&" && "$next_char" == "&" ]]; then
                validate_segment "$segment"
                segment=""
                ((i++))  # Skip next &
            else
                segment+="$char"
            fi
        else
            segment+="$char"
        fi
        
        prev_char="$char"
    done
    
    # Validate the last segment
    if [[ -n "$segment" ]]; then
        validate_segment "$segment"
    fi
}

validate_segment() {
    local segment="$1"
    # Trim leading whitespace
    segment="${segment#"${segment%%[![:space:]]*}"}"
    
    # Skip empty segments
    [[ -z "$segment" ]] && return
    
    for pattern in "${BLOCKED_PATTERNS[@]}"; do
        if echo "$segment" | grep -qE "$pattern"; then
            echo "ERROR: Command blocked by restricted shell: $segment" >&2
            exit 126
        fi
    done
}

# Validate all segments
parse_and_validate_segments "$CMD"

# Execute the command
exec /bin/bash -c "$CMD"
`
