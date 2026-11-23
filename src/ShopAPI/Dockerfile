FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /shop ./cmd/main

FROM alpine:3.23
RUN adduser -D -u 10001 app
USER app
COPY --from=build /shop /usr/local/bin/shop
EXPOSE 8080
ENTRYPOINT ["shop"]
