FROM golang:1.21 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api-gateway ./backend/services/api-gateway/cmd/api-gateway

FROM gcr.io/distroless/base-debian11
WORKDIR /app
COPY --from=builder /app/api-gateway /app/api-gateway
CMD ["/app/api-gateway"]

