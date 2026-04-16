TEST?=$$(go list ./...)
GOFMT_FILES?=$$(find . -name '*.go')
WEBSITE_REPO=github.com/hashicorp/terraform-website
PKG_NAME=nomad

default: build

build: fmtcheck
	go install

test: fmtcheck
	go test -i $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4 -count=1

testacc: fmtcheck
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m -count=1

localtestacc-start-nomad:
	scripts/start-nomad.sh

localtestacc: fmtcheck localtestacc-start-nomad
	-env NOMAD_TOKEN=00000000-0000-0000-0000-000000000000 \
		TF_ACC=1 \
		go test $(TEST) -v $(TESTARGS) -timeout 120m -count=1
	scripts/stop-nomad.sh

vet:
	@echo "go vet ."
	@go vet $$(go list ./...) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

errcheck:
	@sh -c "'$(CURDIR)/scripts/errcheck.sh'"


test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

website:
ifeq (,$(shell which tfplugindocs))
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
endif
	tfplugindocs generate --rendered-website-dir website/docs

.PHONY: build test testacc vet fmt fmtcheck errcheck test-compile website
