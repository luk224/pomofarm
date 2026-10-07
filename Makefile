.PHONY: dev-backend dev-frontend test build

dev-backend:
	cd backend && go run ./cmd/pomofarm

dev-frontend:
	cd frontend && npm run dev

test:
	cd backend && go vet ./... && go test ./...
	cd frontend && npm run lint

build:
	cd backend && go build -o bin/pomofarm ./cmd/pomofarm
	cd frontend && npm run build
