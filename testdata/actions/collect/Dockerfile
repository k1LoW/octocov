FROM golang:1.22.5

ENTRYPOINT ["env", "GOPATH=/go", "GOROOT=", "go", "run", "main.go", "--config", "testdata/octocov_parallel_tests.yml"]
