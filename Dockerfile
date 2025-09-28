# syntax=docker/dockerfile:1

FROM golang:1.25-alpine

# Set the working directory to /app
WORKDIR /build

# Copy the current directory contents into the container at /app
COPY *.go config.yaml template.html ./
COPY go.mod go.sum ./
RUN go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/app
RUN apk add --no-cache ca-certificates

# Run image
FROM scratch
COPY --from=0 /build/app /build/config.yaml /build/template.html /
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Run
CMD ["/app"]
