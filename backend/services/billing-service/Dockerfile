FROM golang:1.23.5 AS builder
ENV CGO_ENABLED=0
ENV GOTOOLCHAIN=local

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /out/billing-service ./backend/services/billing-service/cmd/billing-service

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/billing-service /billing-service
EXPOSE 8083
ENTRYPOINT ["/billing-service"]
