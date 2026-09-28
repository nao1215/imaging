## Contributing as a Developer

### Build, test and lint
imaging is a single Go package. It needs the Go version in `go.mod` or newer.

```shell
make build   # go build ./...
make test    # runs the tests with coverage, writes cover.out and cover.html
make bench   # benchmarks
golangci-lint run ./...
```

There is no lint target in the Makefile. CI runs golangci-lint v2.14.0 with `.golangci.yml` as a required check, so run the same version locally before opening a pull request.

### Pull requests
- When fixing a bug: Create a pull request with a test that fails without the fix.
- When adding a feature: First, propose the feature in an Issue. The pull request must include tests for the new behavior.
- Fuzz targets live in `fuzz_test.go`. `go test` runs them over their seed corpus; `.github/workflows/fuzz.yml` searches with them every night. If a failing input is found, commit it under `testdata/fuzz/<Name>/` together with the fix.

### Reporting bugs and vulnerabilities
- Bugs: Open a GitHub Issue using the bug report template. Include the imaging version, the Go version, and a minimal reproduction.
- Vulnerabilities: Do not open a public Issue. Follow [SECURITY.md](./SECURITY.md).

## Contributing Outside of Coding
The following actions help boost my motivation:

- Giving a GitHub Star
- Promoting the library
- Becoming a GitHub Sponsor
