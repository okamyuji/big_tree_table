FROM golang:1.27-alpine3.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server .

FROM alpine:3.24
RUN apk add --no-cache ca-certificates && adduser -D -H app
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/seed ./seed
USER app
EXPOSE 8080
CMD ["./server"]
