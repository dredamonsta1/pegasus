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
	go install pegasus-web
endef

dev:
	@$(reset)
	@$(fmt)
	@$(build)

default: dev

setup:
	@echo 'Clearing src/github.com...'
	@rm -rf src/github.com/

	@echo 'Getting dependencies...'
	@echo '  github.com/jimmysawczuk/go-config'
	@go get -u github.com/jimmysawczuk/go-config

	@echo '  github.com/gin-gonic/gin'
	@go get -u github.com/gin-gonic/gin
