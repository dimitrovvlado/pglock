GO := go
NAME := pglock
RM := rm

.PHONY: build
build: $(NAME)

$(NAME): $(wildcard *.go) $(wildcard */*.go)
	@echo "+ $@"
	$(GO) build -tags "$(BUILDTAGS)" -o $(NAME) .

.PHONY: test
test: testbed run_tests teardown

.PHONY: run-dev
run-dev:
	@echo "+ $@"
	docker-compose rm -f
	docker-compose -f docker-compose.yaml build
	docker-compose -f docker-compose.yaml up

.PHONY: run_tests
run_tests: testbed
	@echo "+ $@"
	@$(GO) test -v -tags "$(BUILDTAGS) cgo" $(shell $(GO) list ./... | grep -v vendor)

.PHONY: testbed
testbed:
	@echo "+ $@"
	docker-compose up -d --force-recreate
	sleep 5

.PHONY: teardown
teardown:
	@echo "+ $@"
	docker-compose rm --force --stop

.PHONY: clean
clean:
	@echo "+ $@"
	-$(RM) $(NAME)
