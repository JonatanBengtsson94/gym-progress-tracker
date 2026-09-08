FROM golang:1.27

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["go", "run", "./cmd/main.go"]
