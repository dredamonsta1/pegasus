GOPATH := ${PWD}

define reset
	@rm -rf bin
	@rm -rf pkg
endef

define fmt
	@echo 'Running gofmt...';
	find . -type f -name "*.go" | xargs gofmt -w
endef

define build
	@echo 'Building...'

	go install pegasus
endef

dev:
	@$(reset)
	@$(fmt)
	@$(build)

default: dev
