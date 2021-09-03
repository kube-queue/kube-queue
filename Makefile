COMMONENVVAR=GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))
BUILDENVVAR=CGO_ENABLED=0

.PHONY: all
all: build

.PHONY: build
build: build-queue

.PHONY: build-queue
build-queue: fixcodec
	$(COMMONENVVAR) $(BUILDENVVAR) go build -ldflags '-w' -o bin/kube-queue cmd/main.go	

.PHONY: fixcodec
	hack/fix-codec-factory.sh

.PHONY: update-vendor
update-vendor:
	hack/update-vendor.sh

.PHONY: unit-test
unit-test: fixcodec update-vendor
	hack/unit-test.sh

.PHONY: clean
clean:
	rm -rf ./bin
