
# Use an official Golang image as a base
FROM golang:1.22-bullseye

# Set the working directory inside the container
WORKDIR /app

# Install dependencies (including Google Chrome)
RUN apt-get update && apt-get install -y \
  wget \
  gnupg \
  && wget -qO - https://dl.google.com/linux/linux_signing_key.pub | gpg --dearmor > /usr/share/keyrings/google-linux.gpg \
  && echo "deb [signed-by=/usr/share/keyrings/google-linux.gpg] http://dl.google.com/linux/chrome/deb/ stable main" > /etc/apt/sources.list.d/google-chrome.list \
  && apt-get update \
  && apt-get install -y google-chrome-stable \
  && rm -rf /var/lib/apt/lists/*

# Copy the Go source code into the container
COPY . .

# Build the Go app
RUN go build -o bin/momentum-scanner-websocket

# Expose the port your app will listen on
EXPOSE 8080

# Command to run your app
CMD ["bin/momentum-scanner-websocket"]
