# remoton

**DEV STAGE**

(Go) Own remote desktop platform

Commands
  * `remoton-server-cert` generate certificates for secure connections
  * `remoton-server` handle connections between clients and supports
  * `remoton-client-desktop` GUI version for sharing desktop
  * `remoton-support-desktop` GUI version for handling remote desktop


## Release

from....

## Install from source

[Install Go 1.4+](http://golang.org/doc/install) and


### remoton-server

```
go get -u github.com/bit4bit/remoton/cmd/remoton-server
go install github.com/bit4bit/remoton/cmd/remoton-server
```

### remoton-server-cert

```
go get -u github.com/bit4bit/remoton/cmd/remoton-server-cert
go install github.com/bit4bit/remoton/cmd/remoton-server-cert
```

### remoton-client-desktop

```
go get -u github.com/bit4bit/remoton/cmd/remoton-client-desktop
go install github.com/bit4bit/remoton/cmd/remoton-client-desktop
```


### remoton-support-desktop

```
go get -u github.com/bit4bit/remoton/cmd/remoton-support-desktop
go install github.com/bit4bit/remoton/cmd/remoton-support-desktop
```


## Usage

Before start, generate the self-signed certificate and key for server and clients.
Just run:

~~~bash
~$ remoton-server-cert -host="myipserver.org"
~~~

copy **cert.pem** and **key.pem** to restricted folder, **cert.pem**  will be
shared with terminals -client/support-.

Start a server example: 192.168.57.11

~~~bash
~$ remoton-server -listen="192.168.57.11:9934" -cert "path/cert.pem" -key="path/key.pem"-auth-token="public"
~~~

Transfer **cert.pem** their users.

Now you can connect -terminal- a client (share desktop) or support (connect to shared desktop)
we can use the GUI version, just run **remoton-client-desktop** or **remoton-support-desktop**

The will need the **cert.pem** for connect to server.


## POWERED BY

  * [Go v1.4](http://golang.org/)
  * [Xpra](http://www.xpra.org)
  * [Go-Gtk](github.com/mattn/go-gtk)
  * [Log](github.com/Sirupsen/logrus)
  * [HttpRouter](github.com/julienschmidt/httprouter)
  * [Websocket](golang.org/x/net/websocket)