FROM alpine:3.19

# Install bash + ssh client
RUN apk add --no-cache bash openssh

# HOSTS (required): newline-separated host entries.
#   Format: user@host (one per line)
# Only used to SSH into the Hetzner hosts - never passed into VMs.
ENV HOSTS=""

WORKDIR /app

# Copy scripts + SSH users config
COPY entrypoint.sh run-on-remotes.sh reset-ubuntu.sh ssh-users.conf ./

# Make scripts executable
RUN chmod +x entrypoint.sh run-on-remotes.sh reset-ubuntu.sh

ENTRYPOINT ["sh", "./entrypoint.sh"]
CMD ["./reset-ubuntu.sh", "./ssh-users.conf"]
