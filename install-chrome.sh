#!/bin/bash
set -e

# Update package list and install dependencies
apt-get update
apt-get install -y wget gnupg

# Add Google's official GPG key and Chrome's repository
wget -qO - https://dl.google.com/linux/linux_signing_key.pub | gpg --dearmor >/usr/share/keyrings/google-linux.gpg
echo "deb [signed-by=/usr/share/keyrings/google-linux.gpg] http://dl.google.com/linux/chrome/deb/ stable main" >/etc/apt/sources.list.d/google-chrome.list

# Install Google Chrome
apt-get update
apt-get install -y google-chrome-stable
