FROM golang:alpine AS builder

WORKDIR /myapp


COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o . ./cmd/app

EXPOSE 8080

CMD ["./app"]