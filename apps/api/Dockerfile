FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/api ./cmd/api && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/admin ./cmd/admin
FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
RUN mkdir -p /data/uploads && chown app:app /data/uploads
COPY --from=build /out/api /out/admin /usr/local/bin/
USER app
EXPOSE 8080
ENTRYPOINT ["api"]
