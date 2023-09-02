FROM golang:latest

ENTRYPOINT ["env", "GOPATH=/go", "GOROOT=", "go", "run", "main.go", "--config", "testdata/octocov_parallel_tests.yml"]
