FROM golang:1.24.4-alpine
WORKDIR /borm
COPY . .
CMD ["sleep", "100"]