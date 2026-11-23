FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /auth ./cmd/auth

FROM alpine:3.23
RUN adduser -D -u 10001 app
USER app
COPY --from=build /auth /usr/local/bin/auth
EXPOSE 50051
ENTRYPOINT ["auth"]
