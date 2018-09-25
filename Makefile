DOCKER  = docker
VERSION = 1.0.2
REPO    = anuragpaliwal80/boomer

.PHONY: docker-image
docker-image:
	@$(DOCKER) build --rm -t $(REPO):$(VERSION) .
	@$(DOCKER) tag $(REPO):$(VERSION) $(REPO):latest

.PHONY: docker-push
docker-push:
	@$(DOCKER) push $(REPO):$(VERSION) 
	@$(DOCKER) push $(REPO):latest

.PHONY: dev
dev:
	docker build -t boomer-dev -f Dockerfile.dev .
	docker run -it --rm \
	-v $$PWD:/boomer \
	-w /boomer \
	--env-file ./DEV_ENV \
	--net host \
	boomer-dev

.PHONY: build
build:
	docker build -t $(REPO):$(VERSION) .

.PHONY: run
run: build
	docker run -it --rm boomer

.PHONY: release
release: build
	docker push $(REPO):$(VERSION)
