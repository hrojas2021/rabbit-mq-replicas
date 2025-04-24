.PHONY: build send read clean

up_without_workers:
	docker-compose up --build --scale worker1=0 --scale worker2=0 --scale worker3=0  --scale notifier1=0 --scale notifier2=0

up_without_notifiers:
	docker-compose up --build --scale notifier1=0 --scale notifier2=0

up_notifiers:
	docker-compose up --build notifier1 notifier2

send:
	./send_messages.sh

read:
	docker-compose exec worker1 cat /shared/logs/messages.log

clean:
	docker-compose down && docker-compose down --volumes

tail-log:
	docker-compose exec worker1 tail -f /shared/logs/messages.log
