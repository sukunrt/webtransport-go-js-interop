# webtransport-go <-> javascript interop check

Starts a [webtransport-go](https://github.com/quic-go/webtransport-go) server on
port 12345.

Both js creates new session with go server. Both js and go loops receiving and opening streams indefinitely.

## Usage

```bash
go get
QLOGDIR=. go run server.go
# copy printed js code to the browser console or sth like https://codepen.io/pen/?editors=0012
```
