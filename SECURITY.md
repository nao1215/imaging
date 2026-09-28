# Security policy

## Supported versions
Only the latest release of imaging gets fixes, including security fixes. If
you hit an issue on an older version, please reproduce it on the latest release
first.

## Reporting a vulnerability
Report security issues privately, not through public issues or pull requests.

- Email: n.chika156@gmail.com
- Or use the "Report a vulnerability" button on the repository's Security tab.

imaging decodes and processes images that often come from untrusted sources, so
reports about crafted inputs that cause a panic, an unbounded allocation, a hang,
or an out-of-range access in the decoders, the EXIF handling, or the resize and
filter functions are especially useful. Please include enough detail to
reproduce:

- imaging version (the version in your `go.mod`) and Go version
- OS and architecture
- The function you called and what happened
- A minimal reproduction, including the input image if you can share it

## What to expect
imaging is maintained by one developer in spare time, so there is no guaranteed
response time. I will acknowledge the report, confirm the issue, and fix it in a
new release. You will be credited in the release notes unless you prefer to stay
anonymous.
