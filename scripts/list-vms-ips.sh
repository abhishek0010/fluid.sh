#!/bin/bash

# List all active domains
domains=$(virsh list --name | grep -v '^$')

echo "Domain Name   IP Address"
echo "-------------------------"

for domain in $domains; do
  # Fetch the IP address of the domain
  ip=$(virsh domifaddr "$domain" --source agent | awk '/ipv4/ {print $4}' | cut -d'/' -f1)

  # Fallback if no IP from guest agent
  if [ -z "$ip" ]; then
    ip=$(virsh domifaddr "$domain" | awk '/ipv4/ {print $4}' | cut -d'/' -f1)
  fi

  echo -e "$domain\t$ip"
done
