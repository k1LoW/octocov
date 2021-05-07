# octocov

![coverage](docs/coverage.svg)

`octocov` is a tool for collecting code coverage.

Key features of `octocov` are:

- **[Support multiple coverage report formats](#supported-coverage-report-formats).**
- **[Support for even generating coverage report badge](#generate-coverage-report-badge-self).**
- **[Selectable coverage datastore](#store-coverage-report-to-central-datastore).**

## Usage

First, run test with coverage report output.

For example, in case of Go language, add `-coverprofile=coverage.out` option as follows

``` console
$ go test ./... -coverprofile=coverage.out
```

Add `.octocov.yml` ( or `octocov.yml` ) file to your repository, and run `octocov`

``` console
$ octocov
```

### Check for acceptable coverage

By setting `coverage.acceptable:`, the minimum acceptable coverage is specified.

If it is less than that value, the command will exit with exit status `1`.

``` yaml
# .octocov.yml
coverage:
  acceptable: 60%
```

``` console
$ octocov
Error: code coverage is 54.9%, which is below the accepted 60.0%
```

### Generate coverage report badge self.

By setting `coverage.badge:`, generate the coverage report badge self.

``` yaml
# .octocov.yml
coverage:
  badge: docs/coverage.svg
```

You can display the coverage badge without external communication by setting a link to this badge image in README.md, etc.

``` markdown
# mytool

![coverage](docs/coverage.svg)
```

![coverage](docs/coverage.svg)

### Store coverage report to central datastore

By setting `datastore:`, store the coverage reports to central datastore.

#### GitHub

``` yaml
# .octocov.yml
datastore:
  github:
    repository: owner/coverages # central datastore repository
    branch: main                # default: main
    path:                       # default: reports/${GITHUB_REPOSITORY}/report.json
```

#### S3

:construction:

### Central mode

By enabling `central:`, `octocov` acts as a central repository for collecting coverage reports ( [example](example/central/README.md) ).

``` yaml
# .octocov.yml
central:
  enable: true
  root: .          # root directory or index file path of collected coverage reports pages. default: .
  reports: reports # directory where reports are stored. default: reports
  badges: badges   # directory where badges are generated. default: badges
```

When central mode is enabled, other functions are automatically turned off.

## Supported coverage report formats

### Go coverage

**Default path:** `coverage.out`

### LCOV

**Default path:** `coverage/lcov.info`

Support `SF` `DA` only

### SimpleCov

**Default path:** `coverage/.resultset.json`

### Clover

**Default path:** `coverage.xml`

## Install

**deb:**

Use [dpkg-i-from-url](https://github.com/k1LoW/dpkg-i-from-url)

``` console
$ export OCTOCOV_VERSION=X.X.X
$ curl -L https://git.io/dpkg-i-from-url | bash -s -- https://github.com/k1LoW/octocov/releases/download/v$OCTOCOV_VERSION/octocov_$OCTOCOV_VERSION-1_amd64.deb
```

**RPM:**

``` console
$ export OCTOCOV_VERSION=X.X.X
$ yum install https://github.com/k1LoW/octocov/releases/download/v$OCTOCOV_VERSION/octocov_$OCTOCOV_VERSION-1_amd64.rpm
```

**apk:**

Use [apk-add-from-url](https://github.com/k1LoW/apk-add-from-url)

``` console
$ export OCTOCOV_VERSION=X.X.X
$ curl -L https://git.io/apk-add-from-url | sh -s -- https://github.com/k1LoW/octocov/releases/download/v$OCTOCOV_VERSION/octocov_$OCTOCOV_VERSION-1_amd64.apk
```

**homebrew tap:**

```console
$ brew install k1LoW/tap/octocov
```

**manually:**

Download binary from [releases page](https://github.com/k1LoW/octocov/releases)

**go get:**

```console
$ go get github.com/k1LoW/octocov
```

**docker:**

```console
$ docker pull ghcr.io/k1low/octocov:latest
```
