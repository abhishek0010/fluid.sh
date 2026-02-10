FROM alpine:3.19

# Install bash + ssh client + sshpass for password-based auth
RUN apk add --no-cache bash openssh sshpass

WORKDIR /app

# Copy scripts + host list + SSH users config
COPY run-on-remotes.sh hosts.txt reset-ubuntu.sh ssh-users.conf ./

# Make scripts executable
RUN chmod +x run-on-remotes.sh reset-ubuntu.sh

# SSH credentials for connecting to remote hosts
# SSH_USER overrides the user in hosts.txt (default: use hosts.txt as-is)
# SSH_PASSWORD enables sshpass-based password authentication
ENV SSH_USER=""
ENV SSH_PASSWORD=""

# Default command
ENTRYPOINT ["bash", "./run-on-remotes.sh", "./hosts.txt", "./reset-ubuntu.sh", "./ssh-users.conf"]
