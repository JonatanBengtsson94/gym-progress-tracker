FROM golang:1.27

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN go build -ldflags "-X main.version=${VERSION}" -o /usr/local/bin/server ./cmd

CMD ["server"]
