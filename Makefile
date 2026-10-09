.PHONY: dev-backend dev-frontend test test-time build

dev-backend:
	cd backend && go run ./cmd/pomofarm

dev-frontend:
	cd frontend && npm run dev

test:
	cd backend && go vet ./... && go test ./...
	cd frontend && npm run lint && npm test

# Suite de pruebas de tiempo (docs/qa/tiempo.md). Con E2E=1 ejecuta también los scripts de navegador.
test-time:
	python3 tools/time_suite.py $(if $(E2E),--e2e,)

build:
	cd backend && go build -o bin/pomofarm ./cmd/pomofarm
	cd frontend && npm run build
