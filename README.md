# Lock N Load

Go app development auto/hot reload

### Manual
Usage:
  locknload [flags]

Flags:

  -b, --build string    target build entry that will be rebuilt every time there is file update (default "*.go")

  -h, --help            help for locknload

  -o, --output string   where to store the temporary built app (default "/tmp/locknload/app")

  -t, --target string   target directory being observed. (default "./app")

### Examples

```
// This will trigger a build and start the app. Subsequent builds will trigger rebuild and restart.
locknload -t /path/to/working/directory -b entry_file.go
```