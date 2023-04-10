# Base image
FROM golang:latest

# Set the working directory to /app
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . /app

# Compile the Go program
RUN go build -o main .

# Expose port 8080
EXPOSE 8080

# Set the command to run the executable
CMD ["./main"]

