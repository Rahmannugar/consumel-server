# syntax=docker/dockerfile:1

FROM golang:1.26-alpine

WORKDIR /workspace

RUN apk add --no-cache ca-certificates

COPY . .

EXPOSE 8080

CMD ["go", "run", "./cmd/api"]
