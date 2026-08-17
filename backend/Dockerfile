FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /scenic-api .

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /scenic-api /usr/local/bin/scenic-api

EXPOSE 8741

CMD ["scenic-api"]
