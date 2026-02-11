#!/bin/sh

# Set up SSH key from Render secret mount (or any mounted key).
# If the key file is missing, warn but continue - allows local testing.

mkdir -p ~/.ssh
chmod 700 ~/.ssh

if [ -f /etc/secrets/SSH_PRIVATE_KEY ]; then
    cp /etc/secrets/SSH_PRIVATE_KEY ~/.ssh/id_ed25519
    chmod 600 ~/.ssh/id_ed25519
else
    echo "[WARN] /etc/secrets/SSH_PRIVATE_KEY not found - SSH key auth will not be available"
fi

exec bash ./run-on-remotes.sh "$@"
