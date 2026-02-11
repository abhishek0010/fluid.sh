FROM alpine:3.19

# Install bash + ssh client + sshpass (for password-based SSH)
RUN apk add --no-cache bash openssh sshpass

# SSH credentials for connecting to remote hosts.
# When set, run-on-remotes.sh uses sshpass for password-based auth instead of key-based auth.
# Only used to connect to the Hetzner hosts - never passed into VMs or cloud-init.
# Usage: docker run -e SSH_USER=root -e SSH_PASSWORD=secret ...
ENV SSH_USER=""
ENV SSH_PASSWORD=""

WORKDIR /app

# Copy scripts + host list + SSH users config
COPY run-on-remotes.sh hosts.txt reset-ubuntu.sh ssh-users.conf ./

# Make scripts executable
RUN chmod +x run-on-remotes.sh reset-ubuntu.sh

# Default command
ENTRYPOINT ["bash", "./run-on-remotes.sh", "./hosts.txt", "./reset-ubuntu.sh", "./ssh-users.conf"]
