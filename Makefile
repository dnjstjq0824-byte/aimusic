.PHONY: backend-build backend-vet backend-test web-install web-typecheck web-test web-build mobile-analyze scripts-check all e2e local-up

backend-build:
	cd backend && go build ./...

backend-vet:
	cd backend && go vet ./... && test -z "$$(gofmt -l .)"

backend-test:
	cd backend && go test ./...

web-install:
	cd web && npm ci

web-typecheck:
	cd web && npm run typecheck

web-test:
	cd web && npm run test -- --run

web-build:
	cd web && npm run build

mobile-analyze:
	cd mobile && flutter analyze

scripts-check:
	bash -n scripts/test-e2e.sh
	bash -n scripts/start-local.sh

all: backend-build backend-vet backend-test web-typecheck web-test web-build mobile-analyze scripts-check

local-up:
	./scripts/start-local.sh --build

e2e:
	./scripts/test-e2e.sh
