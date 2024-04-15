# Build

## Prerequisites

```bash
# macOS
brew install go `# golang compiler`
brew install colima `# docker substitute`
colima start --mount-type 9p
```

## Unit/Functional Tests

```bash
go install gotest.tools/gotestsum@latest `# nicer test runner`
gotestsum -f testname
```

## Build

```bash
./output/build.sh
```

## Release

[for maintainer] do not create git tags manually, update the changelog and the script will tag based on it.

```bash
./output/release.sh
```