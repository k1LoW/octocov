PKG = github.com/k1LoW/octocov
COMMIT = $$(git describe --tags --always)
OSNAME=${shell uname -s}
ifeq ($(OSNAME),Darwin)
	DATE = $$(gdate --utc '+%Y-%m-%d_%H:%M:%S')
else
	DATE = $$(date --utc '+%Y-%m-%d_%H:%M:%S')
endif

export GO111MODULE=on
export CGO_ENABLED=1

BUILD_LDFLAGS = -X $(PKG).commit=$(COMMIT) -X $(PKG).date=$(DATE)

default: test

ci: depsdev test test_no_coverage sec

test:
	go test ./... -coverprofile=coverage.out -covermode=count

test_central: build
	./octocov --config testdata/octocov_central.yml

test_no_coverage: build
	./octocov --config testdata/octocov_no_coverage.yml

test_collect_metrics: build
	./octocov --config testdata/octocov_parallel_tests.yml

sec:
	gosec ./...

lint:
	golangci-lint run ./...

build:
	go build -ldflags="$(BUILD_LDFLAGS)"

coverage: build
	./octocov

bqdoc:
	cd docs/bq && tbls doc -f

depsdev:
	go install github.com/Songmu/ghch/cmd/ghch@v0.10.2
	go install github.com/Songmu/gocredits/cmd/gocredits@v0.2.0
	go install github.com/securego/gosec/v2/cmd/gosec@latest

prerelease:
	git pull origin main --tag
	go mod tidy
	ghch -w -N ${VER}
	gocredits -skip-missing -w
	cat _EXTRA_CREDITS >> CREDITS
	git add CHANGELOG.md CREDITS go.mod go.sum
	git commit -m'Bump up version number'
	git tag ${VER}

release:
	git push origin main --tag
	goreleaser --rm-dist

.PHONY: default test
