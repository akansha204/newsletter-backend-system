.PHONY: infra-up infra-down infra-logs

infra-up:
	cd deploy && docker compose up -d

infra-down:
	cd deploy && docker compose down

infra-logs:
	cd deploy && docker compose logs -f

infra-clean:
	cd deploy && docker compose down -v