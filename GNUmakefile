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
# vendor
###############################################################################

vendor:
	@dep ensure

vendor-status:
	@dep status

.PHONY: vendor vendor-status

###############################################################################
# testing
###############################################################################

test: fmtcheck
	go test -i $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=2m -parallel=4

testacc: fmtcheck
	@TF_ACC=1 go test ./acsengine -v -timeout 15h

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

coverage:
	@scripts/coverage.sh --codecov
 
.PHONY: test testacc lint fmtcheck errcheck test-compile coverage

###############################################################################
# CI tests
###############################################################################

cluster-create:
	TF_ACC=1 go test ./acsengine -v -run createBasic -timeout 3h
	TF_ACC=1 go test ./acsengine -v -run createVersion10AndAbove -timeout 3h

cluster-scale:
	TF_ACC=1 go test ./acsengine -v -run scaleUpDown -timeout 5h
	TF_ACC=1 go test ./acsengine -v -run scaleDownUp -timeout 5h

cluster-upgrade:
	TF_ACC=1 go test ./acsengine -v -run upgradeMultiple -timeout 5h
	TF_ACC=1 go test ./acsengine -v -run upgradeVersion10AndAbove -timeout 5h

cluster-update-scale:
	TF_ACC=1 go test ./acsengine -v -run updateScaleUpUpgrade -timeout 5h
	TF_ACC=1 go test ./acsengine -v -run updateScaleDownUpgrade -timeout 5h

cluster-update-upgrade:
	TF_ACC=1 go test ./acsengine -v -run updateUpgrade -timeout 5h

cluster-update-tags:
	TF_ACC=1 go test ./acsengine -v -run updateTags -timeout 5h

cluster-data:
	TF_ACC=1 go test ./acsengine -v -run DataSource -timeout 5h

cluster-import:
	TF_ACC=1 go test ./acsengine -v -run importBasic -timeout 5h

cluster-windows:
	TF_ACC=1 go test ./acsengine -v -run windowsCreate -timeout 5h

.PHONY: cluster-create cluster-scale cluster-upgrade cluster-update-scale cluster-update-upgrade cluster-update-tags cluster-data cluster-import

###############################################################################
# key vault for CI tests
###############################################################################

keyvault-apply:
	@scripts/keyvault.sh --tfapply

keyvault-destroy:
	@scripts/keyvault.sh --tfdestroy

.PHONY: keyvault-apply keyvault-destroy