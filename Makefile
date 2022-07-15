.PHONY: restart
restart:
	docker compose down --remove-orphans && docker compose build && DD_API_KEY=$(DD_API_KEY) docker compose up -d

.PHONY: stop
stop:
	docker compose down --remove-orphans

.PHONY: ping-a
ping-a:
	curl http://localhost:8001/ping

.PHONY: ping-ab
ping-ab:
	curl http://localhost:8001/ping?forward=true