FROM golang:1.21-alpine

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN go mod download || true

COPY . .

RUN go build -o main .

EXPOSE 8741

CMD ["./main"]
