FROM golang:1.23.5 AS builder
ENV CGO_ENABLED=0
ENV GOTOOLCHAIN=local

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /out/sessions-service ./backend/services/sessions-service/cmd/sessions-service

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/sessions-service /sessions-service
EXPOSE 8082
ENTRYPOINT ["/sessions-service"]
