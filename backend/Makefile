.PHONY: run build tidy up down logs psql test

run:
	go run ./cmd/api

build:
	go build -o ./out/api ./cmd/api

tidy:
	go mod tidy

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f api

psql:
	docker compose exec postgres psql -U blazeaid -d blazeaid

test:
	go test ./...
