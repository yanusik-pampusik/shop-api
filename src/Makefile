.PHONY: test vet build integration proto swagger up down

test:
	cd AuthService && go test -race ./...
	cd ShopAPI && go test -race ./...

vet:
	cd AuthService && go vet ./...
	cd ShopAPI && go vet ./...

build:
	mkdir -p bin
	cd AuthService && go build -o ../bin/auth ./cmd/auth
	cd ShopAPI && go build -o ../bin/shop ./cmd/main

integration: build
	python3 scripts/integration.py

proto:
	sh scripts/generate-proto.sh

swagger:
	cd ShopAPI && go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/main/main.go -o docs

up:
	docker compose up --build -d

down:
	docker compose down
