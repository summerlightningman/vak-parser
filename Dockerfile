# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /vak-parser .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        poppler-utils \
        tesseract-ocr \
        tesseract-ocr-eng \
        tesseract-ocr-rus \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /vak-parser /app/vak-parser

RUN useradd -r -u 1000 appuser \
    && mkdir -p /data \
    && chown appuser:appuser /data

USER appuser

ENV DB_PATH=/data/vak.db

VOLUME /data

CMD ["/app/vak-parser"]
