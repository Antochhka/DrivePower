FROM golang:1.23.5 AS builder
ENV CGO_ENABLED=0
ENV GOTOOLCHAIN=local

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /out/auth-service ./backend/services/auth-service/cmd/auth-service

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/auth-service /auth-service
EXPOSE 8085
ENTRYPOINT ["/auth-service"]
