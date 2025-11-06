FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY third_party ./third_party
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o csms ./main.go

FROM alpine:3.19
RUN adduser -D -H csms
WORKDIR /app
COPY --from=builder /app/csms .
EXPOSE 8080
USER csms
CMD ["./csms"]
