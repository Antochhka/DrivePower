FROM golang:1.23.5 AS builder
ENV CGO_ENABLED=0
ENV GOTOOLCHAIN=local

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /out/ocpp-server ./backend/services/ocpp-server/cmd/ocpp-server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/ocpp-server /ocpp-server
EXPOSE 8081
ENTRYPOINT ["/ocpp-server"]
