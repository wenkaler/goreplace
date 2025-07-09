# goreplace

[![Go](https://img.shields.io/badge/go-1.20+-blue)](https://golang.org)
[![License](https://img.shields.io/github/license/wenkaler/goreplace)](https://github.com/wenkaler/goreplace)

**goreplace** is a CLI tool that helps you replace Go module dependencies with local paths by modifying the `go.mod` file.

## Description

This utility scans your `go.mod`, finds dependencies matching a given partial package name, and replaces them with a local path pointing to your filesystem (typically under `$GOPATH/src/`).

It's useful during development when you want to test changes in a dependency without publishing it.

---

## Installation

You can install **goreplace** using `go install`:

```bash
go install github.com/wenkaler/goreplace@latest
```

Or build from source:

```bash
git clone https://github.com/wenkaler/goreplace.git
cd goreplace
go build -o goreplace
```

---

## Usage

```bash
Usage: goreplace <partial-package-name>
```

### Example:

```bash
goreplace proto
```

The tool will:
1. Search for dependencies containing "proto" in `go.mod`
2. Let you select one if multiple matches are found
3. Try to locate the local copy of the module
4. Add a `replace` directive in `go.mod` pointing to the local path

---

## Flags

| Flag        | Short | Description           |
|-------------|-------|------------------------|
| `--help`    | `-h`  | Show help              |
| `--version` |       | Show version number    |

---

## Requirements

- Go 1.16 or newer
- `GOPATH` set up correctly
- A valid `go.mod` file in current directory

