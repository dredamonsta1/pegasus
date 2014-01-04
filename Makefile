GOPATH := ${PWD}

define reset
	# @rm -f src/static/compiled*.go
	@rm -rf bin
	# @rm -rf deploy
	@rm -rf pkg
	@mkdir -p bin
	# @mkdir -p deploy
	@mkdir -p pkg
endef

define fmt
	@echo 'Running gofmt...';
	find . -type f -name "*.go" | xargs gofmt -w
endef

define build
	@echo 'Building...'

	GOOS=$(OS) GOARCH=$(ARCH) go install pegasus
endef

dev:
	@$(reset)
	@$(fmt)
	@$(build)

default: dev
