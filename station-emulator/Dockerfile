FROM rust:1.82 as builder
WORKDIR /usr/src/app
COPY Cargo.toml Cargo.lock ./
COPY src ./src

RUN cargo build --release

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        libnss3 \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m emulator
WORKDIR /usr/src/app
COPY --from=builder /usr/src/app/target/release/station-emulator ./station-emulator
USER emulator
CMD ["./station-emulator"]
