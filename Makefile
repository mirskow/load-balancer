.PHONY: build run down stop test

build:
	docker-compose build

run:
	docker-compose up

down:
	docker-compose down --volumes --remove-orphans

stop:
	docker-compose stop

test:
	docker run -d --name redis-test -p 6380:6379 redis

	cd load-balancer/tests && go test -bench=. -race

	docker rm -f redis-test
	
