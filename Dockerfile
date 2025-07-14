FROM golang:1.24.4-alpine
WORKDIR /borm
COPY go.mod ./
RUN go mod download
COPY . .
CMD ["go", "run", "cmd/sleep/sleep.go"]