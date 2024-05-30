# webtransport-send-lots-of-data

Starts a [webtransport-go](https://github.com/quic-go/webtransport-go) server on
port 12345.

The server accepts incoming WebTransport sessions, waits for a bidirectional
stream to be opened on the session, writes 256MB of data into it an closes the
stream.

The client connects to the server, opens a bidirectional stream, reads all the
data from it an expects to receive all of the data.

It works on Chrome, fails on Firefox.

## Usage

Clone this repo then:

* Install `go1.20` or later
* Install go deps with `go get`

## Node.js

* Run `npm start`
* You should see something like:

```console
% npm start

> go-webtransport-readable-never-ends@1.0.0 start
> node index.js

SERVER start
SERVER ready

Paste the following code into https://codepen.io/pen/?editors=0012 or simmilar:

(async function main ()  {
  console.info('CLIENT create session')
... more code here
```

* Paste the JS code into a https://codepen.io/pen/?editors=0012 or otherwise run it in a browser
* Ensure the browser console is visible
* See the browser output:

```
// lots of output
"bytes" 268362464
"bytes" 268369920
"bytes" 268406352
"bytes" 268427592
```

If the stream ends successfully, you'll see:

```
"CLIENT read stream finished"
"CLIENT received" 268435456 "bytes of 268435456"
```

If it doesn't, you won't.
