TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
WEBSITE_REPO=github.com/hashicorp/terraform-website
PKG_NAME=acsengine

###############################################################################
# build
###############################################################################

default: build

build: fmtcheck generate-all
	go install

###############################################################################
# generate for acs-engine
###############################################################################

prereqs:
	go get github.com/golang/dep/cmd/dep
	go get github.com/jteeuwen/go-bindata/...

generate-templates: prereqs
	go generate ./vendor/github.com/Azure/acs-engine/pkg/acsengine

generate-translations: prereqs
	go generate ./vendor/github.com/Azure/acs-engine/pkg/i18n

generate-all: generate-templates generate-translations

###############################################################################
# testing
###############################################################################

test: fmtcheck
	unset TF_ACC
	go test -i $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=90s -parallel=4

testacc: fmtcheck
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 180m

debugacc: fmtcheck
	TF_ACC=1 dlv test $(TEST) --headless --listen=:2345 --api-version=2 -- -test.v $(TESTARGS)

lint:
	gometalinter ./acsengine/... --deadline 100s

vet:
	@echo "go vet ."
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

fmt:
	gofmt -w $(GOFMT_FILES)

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

###############################################################################
# vendor
###############################################################################

vendor-status:
	@dep status

.PHONY: build test testacc vet fmt fmtcheck errcheck vendor-status test-compile

