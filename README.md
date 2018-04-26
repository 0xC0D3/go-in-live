# go-in-live
Auto compiler and runner with files watcher which keeps running the latest code version.

## Previous requirements

- [**Dep**](https://golang.github.io/dep/) is required to manage dependencies.

## Setup

**Windows example:**
```shell
go get github.com\0xC0D3\go-in-live
cd %gopath%\src\github.com\0xC0D3\go-in-live
dep ensure
go install github.com\0xC0D3\go-in-live
```

## Usage of go-in-live

```
-build string
    Custom build command. (default "go build -o $1")
-i    Redirect input to the executable.
-run string
    Custom run command. (default "$1")
-watch string
    Comma separated paths to watch, in case you want to watch all files
    inside a directory, use the ".\< dir >/*" format. (default "./.c0d3v")
```
