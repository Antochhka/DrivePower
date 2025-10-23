FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o csms ./main.go

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/csms .
EXPOSE 8080
USER nonroot:nonroot
CMD ["./csms"]