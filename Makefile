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

.PHONY: ping-ba
ping-ba:
	curl http://localhost:8002/ping?forward=true