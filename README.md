# Gofs

A simple static file server inspired by [dufs](https://github.com/sigoden/dufs).
`gofs` uses zero JavaScript and so it can function properly anywhere (for example, in a browser of e-reader device).

![image](https://github.com/user-attachments/assets/290ba8b9-de77-43e3-858e-e4cb03ed189a)

## Install

### With go
```bash
go install github.com/ndtoan96/gofs@latest
```

### Pre-built binary
You can download pre-built binary in the [release page](https://github.com/ndtoan96/gofs/releases)

## Features

- [x] Serve static files
- [x] New folder
- [x] Delete files/folders
- [x] Archive
- [x] Rename
- [x] Upload files
- [x] Download
- [x] Copy/Paste
- [x] Edit
- [x] Support https
- [ ] Searching
- [ ] Drag and drop
- [ ] Sorting
- [ ] Serve index.html

## Usage
```
Usage of C:\Users\ASUS\go\bin\gofs.exe:
      --tsl-cert string   Path to an SSL/TLS certificate to serve with HTTPS
      --tsl-key string    Path to an SSL/TLS certificate's private key
  -d, --dir string          Directory to serve (default ".")
  -h, --host string         Host address to listen (default "[::]")
  -p, --port int            Port to listen (default 8080)
  -w, --write               Allow write access
```

## Example
Serve current directory in read-only mode
```bash
gofs
```

Serve current directory in write mode (allow copy, paste, rename, edit, delete,...)
```bash
gofs -w
```

Serve directory `xxx`
```bash
gofs xxx
```

Use a different port (default is 8080)
```bash
gofs -p 7777
```
