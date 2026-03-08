FROM golang:1.25

WORKDIR /app

RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN go install github.com/air-verse/air@latest
RUN #air init

COPY go.mod go.sum ./

RUN go mod download

CMD ["air", "-c", ".air.toml"]