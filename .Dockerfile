
FROM golang:1.20-bullseye

# Install dependencies
RUN apt-get update && apt-get install -y \
  wget \
  gnupg

# Add Google's official GPG key and install Chrome
RUN wget -qO - https://dl.google.com/linux/linux_signing_key.pub | gpg --dearmor > /usr/share/keyrings/google-linux.gpg
RUN echo "deb [signed-by=/usr/share/keyrings/google-linux.gpg] http://dl.google.com/linux/chrome/deb/ stable main" > /etc/apt/sources.list.d/google-chrome.list
RUN apt-get update && apt-get install -y google-chrome-stable

# Set work directory and build the Go app
WORKDIR /app
COPY . .
RUN go build -o app .

# Start the application
CMD ["./app"]
