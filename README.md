# octocov

![coverage](docs/coverage.svg)

`octocov` is a tool for collecting code coverage.

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

```
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

```
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

### Store coverage report to datastore

By setting `datastore:`, store the coverage reports.

#### GitHub

```
datastore:
  github:
    repository: owner/repo # datastore repository
    branch: main # default: main
    path: # default: report/${GITHUB_REPOSITORY}.json
```

#### S3

:construction:

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
