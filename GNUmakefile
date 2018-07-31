TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
PKG_NAME=acsengine
undefine TF_ACC

###############################################################################
# build
###############################################################################

default: build

build: fmtcheck prereqs generate-all
	go install
	
HAS_DEP := $(shell command -v dep;)
HAS_GIT := $(shell command -v git;)
HAS_GOMETALINTER := $(shell command -v gometalinter;)

prereqs:
ifndef HAS_DEP
	go get -u github.com/golang/dep/cmd/dep
endif
	go get github.com/jteeuwen/go-bindata/...
ifndef HAS_GIT
	$(error you must install Git)
endif
ifndef HAS_GOMETALINTER
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
endif

.PHONY: build prereqs

###############################################################################
# generate for acs-engine
###############################################################################

generate-templates:
	go generate ./vendor/github.com/Azure/acs-engine/pkg/acsengine

generate-translations:
	go generate ./vendor/github.com/Azure/acs-engine/pkg/i18n

generate-all: generate-templates generate-translations

.PHONY: generate-templates generate-translations

###############################################################################
# testing
###############################################################################

# do I want all of these?

test: fmtcheck
	go test -i $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=120s -parallel=4

testacc: fmtcheck
	# TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 10h
	TF_ACC=1 go test ./acsengine -v -run TestAccACSEngine -timeout 15h # I'm thinking about running this instead

debugacc: fmtcheck
	TF_ACC=1 dlv test $(TEST) --headless --listen=:2345 --api-version=2 -- -test.v $(TESTARGS)

lint:
	gometalinter ./acsengine/... --disable-all \
		--enable=vet --enable=gotype --enable=deadcode --enable=golint --enable=varcheck \
		--enable=structcheck --enable=errcheck --enable=ineffassign --enable=unconvert --enable=goconst \
		--enable=misspell --enable=goimports --enable=gofmt --deadline 100s
# I would like to add in dupl, vetshadow, and megacheck eventually, maybe gocyclo, gosec, and others

fmtcheck:
	@sh "$(CURDIR)/scripts/gofmtcheck.sh"

errcheck:
	@sh "$(CURDIR)/scripts/errcheck.sh"

test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

# figure out coverage
# coverage:

.PHONY: test testacc lint fmtcheck errcheck test-compile

###############################################################################
# vendor
###############################################################################

vendor:
	@dep ensure

vendor-status:
	@dep status

.PHONY: vendor vendor-status
