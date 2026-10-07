FROM golang:1.27 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION}" -o /out/server ./cmd
RUN CGO_ENABLED=0 go build -o /out/createuser ./cmd/createuser

FROM alpine:3.24

LABEL org.opencontainers.image.source=https://github.com/JonatanBengtsson94/gym-progress-tracker

COPY --from=build /out/ /usr/local/bin/

USER nobody

CMD ["server"]
