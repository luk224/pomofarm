FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/pomofarm ./cmd/pomofarm

FROM alpine:3.21
RUN adduser -D -u 10001 pomofarm && mkdir -p /data /backups && chown pomofarm /data /backups
COPY --from=build /out/pomofarm /usr/local/bin/pomofarm
USER pomofarm
ENV POMOFARM_ADDR=:8080 POMOFARM_DB=/data/pomofarm.db POMOFARM_BACKUPS=/backups
EXPOSE 8080
ENTRYPOINT ["pomofarm"]
