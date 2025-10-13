FROM golang:1.25-alpine AS builder
WORKDIR /build
ENV version_env="1.0.0"
COPY . .
RUN apk update && apk upgrade && apk add --no-cache gcc musl-dev
RUN go build -ldflags="-X 'main.version=$version_env'" -o /main .
WORKDIR cmd/migrate
RUN CGO_ENABLED=1 go build -o /migrate .

FROM alpine:3.22

RUN apk add --no-cache tzdata
RUN cp /usr/share/zoneinfo/Europe/Moscow /etc/localtime
RUN echo "Europe/Moscow" > /etc/timezone

COPY --from=builder migrate /migrate/migrate
COPY cmd/migrate/config.yml /migrate/config.yml
RUN chmod 666 /migrate/config.yml

ARG UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    appuser

USER appuser

WORKDIR /app
ENV app_name_env="t-invest-bot"
COPY --from=builder main /app/$app_name_env
COPY /conf/config.yml /app/conf/config.yml
ENTRYPOINT /app/$app_name_env