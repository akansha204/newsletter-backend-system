.PHONY: infra-up infra-down infra-logs migrate

infra-up:
	cd deploy && docker compose up -d

infra-down:
	cd deploy && docker compose down

infra-logs:
	cd deploy && docker compose logs -f

infra-clean:
	cd deploy && docker compose down -v

migrate:
	docker exec -i newsletter_postgres psql -U newsletter -d newsletter_db < migrations/001_create_subscribers.sql
	docker exec -i newsletter_postgres psql -U newsletter -d newsletter_db < migrations/002_create_newsletter_sends.sql