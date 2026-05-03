FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /cgate ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates docker-cli
COPY --from=builder /cgate /usr/local/bin/cgate

EXPOSE 8000
ENTRYPOINT ["cgate"]
