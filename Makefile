.PHONY: init up down generate test admin
init:
	./scripts/init-env.sh
up: init
	docker compose up --build -d
# Does not remove persistent volumes.
down:
	docker compose down
generate:
	cd apps/api && sqlc generate
test:
	cd apps/api && go test -race ./...
	cd apps/web && npm run typecheck && npm test
admin:
	docker compose run --rm -e ADMIN_EMAIL -e ADMIN_NAME -e ADMIN_PASSWORD --entrypoint admin api
