FROM rust:1.76 as builder

WORKDIR /usr/src/app
COPY Cargo.toml Cargo.lock ./
COPY src ./src

RUN cargo build --release

FROM debian:bookworm-slim
RUN useradd -m emulator
WORKDIR /app
COPY --from=builder /usr/src/app/target/release/station-emulator ./station-emulator
USER emulator
CMD ["./station-emulator"]
