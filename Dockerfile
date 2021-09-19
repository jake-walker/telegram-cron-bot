FROM golang:1.16-alpine

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN go build -o main .

LABEL org.opencontainers.image.vendor="Jake Walker"
LABEL org.opencontainers.image.authors="Jake Walker"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.title="Telegram Cron Bot"
LABEL org.opencontainers.image.description="A Telegram bot for creating scheduled tasks"
LABEL com.centurylinklabs.watchtower.enable="true"

ENV BOT_CONFIG_DIRECTORY="/config"

CMD ["/app/main"]
