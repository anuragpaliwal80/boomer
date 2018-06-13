.PHONY: dev build run

default: dev

dev:
	docker build -t boomer-dev -f Dockerfile.dev .
	docker run -it --rm \
	-v $$PWD:/go/src/github.com/abhisheknsit/boomer \
	-w /go/src/github.com/abhisheknsit/boomer \
	--env-file ./DEV_ENV \
	--net host \
	boomer-dev

build:
	docker build -t boomer .

run: build
	docker run -it --rm boomer
