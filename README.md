# go-fz

A small command-line tool that walks a directory tree and reports the
largest files and the largest directories it finds, up to a configurable
depth.

Directory sizes are always computed from the **entire** subtree — the
`-depth` flag only controls which directories are *listed*, not what gets
counted. So `a/b`'s reported size includes everything under `a/b/c`, `a/b/d`,
etc., even if those deeper directories themselves are past the depth cutoff.

## Install / build

Requires Go 1.24+ (developed against 1.26).

```sh
git clone <this repo>
cd go-fz
make build      # produces ./bin/go-fz
```

Or install it onto your `$GOPATH/bin`:

```sh
make install
```

## Usage

```sh
go-fz [flags] [path]
```

If `path` is omitted, the current directory is used.

### Flags

| Flag        | Default | Description                                                        |
|-------------|---------|----------------------------------------------------------------------|
| `-depth`    | `3`     | Max depth (relative to `path`) of directories to report              |
| `-top`      | `10`    | Number of results to show in each list                               |
| `-files`    | `true`  | Show the largest-files list                                          |
| `-dirs`     | `true`  | Show the largest-directories list                                    |
| `-exclude`  | `.git`  | Comma-separated directory names to skip entirely (e.g. `node_modules`) |
| `-human`    | `true`  | Print human-readable sizes (KiB/MiB/GiB) instead of raw byte counts   |

### Examples

Largest files/dirs in the current directory, default depth and top-10:

```sh
go-fz
```

Look at `/var/log`, only 2 levels deep, top 5 of each, skip `archive` dirs:

```sh
go-fz -depth 2 -top 5 -exclude ".git,archive" /var/log
```

Just the largest files, raw byte counts, no directory list:

```sh
go-fz -dirs=false -human=false /path/to/scan
```

## How it works

1. A single `filepath.WalkDir` pass collects every regular file (for the
   files list) and registers every directory along with its depth relative
   to the scan root.
2. Each file's size is added to its immediate parent directory.
3. Directories are then aggregated bottom-up: the pre-order directory list
   from the walk is processed in reverse, which guarantees every descendant
   is folded into its parent before that parent is folded into *its*
   parent. The result is an accurate cumulative size for every directory in
   the tree, computed in a single pass.
4. Both lists are sorted descending by size and truncated to `-top` entries;
   the directories list is additionally filtered to `depth <= -depth`.

Permission errors and unreadable directories are logged to stderr and
skipped rather than aborting the whole scan.

## Development

```sh
make build       # build ./bin/go-fz
make run ARGS="-depth 1 ."   # build and run with args
make test        # go test ./...
make vet         # go vet ./...
make fmt         # gofmt -w .
make fmt-check   # verify formatting in CI
make clean       # remove the built binary
```
