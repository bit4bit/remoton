![Logo Remoton](https://cloud.githubusercontent.com/assets/1474826/8950994/543baebc-358e-11e5-886c-d4c440d3417f.png)

**DEV STAGE**

(Go) Own secure remote desktop multi-platform, own platform for sharing your desktop with software libre

  * do support
  * get desktop access to remote user
  * upload/download files
  * chat
  
**Remoton desktop** it's proof concept of **Remoton Library**

Commands
  * `remoton-server-cert` generate certificates for secure connections
  * `remoton-server` handle connections between clients and supports
  * `remoton-client-desktop` GUI version for sharing desktop
  * `remoton-support-desktop` GUI version for handling remote desktop

## Help

Email me bit4bit@riseup.net

## Contributing

What ever you like

## Release

See [Releases](https://github.com/bit4bit/remoton-release/releases)

## Install from release

Please check at [Releases](https://github.com/bit4bit/remoton-release/releases)

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


## TODO

  * upnp/pmp only works on GNU/Linux, windows xpra --auth=file not work

## POWERED BY

  * [Go v1.4](http://golang.org/)
  * [Xpra](http://www.xpra.org)
  * [Go-Gtk](github.com/mattn/go-gtk)
  * [Log](github.com/Sirupsen/logrus)
  * [HttpRouter](github.com/julienschmidt/httprouter)
  * [Websocket](golang.org/x/net/websocket)

# Library

Remoton it's a library for building programmatically tunnels.
See [Doc](http://godoc.org/github.com/bit4bit/remoton)

  * Now only support Websocket -Binary- and TCP.


## Listener

You can listen inbound connections on remoton server
~~~go
	import "github.com/bit4bit/remoton"
	....
	
	
	rclient := remoton.Client{Prefix: "/remoton", TLSConfig: &tls.Config{
		InsecureSkipVerify: true,
	}}
	session, err := rclient.NewSession("https://miserver.com:9934", "public")
	if err != nil {
		log.Fatal(err)
	}
	defer session.Destroy()
	
	//now can create a listener for every service you want
	//example
	listener := session.Listen("chat")
	go func(){
		 for {
			conn, err := listener.Accept()
			//now you use conn -net.Conn-
		 }
	}()
	
	listener = session.Listen("rpc")
	//or use it example RPC
	srvRpc := rpc.NewServer()
	srvRpc.Register(&Api)
	go srvRpc.Accept(listener)
~~~

## Dial

You can dial a active session.
~~~go
    import "github.com/bit4bit/remoton"

	rclient := remoton.Client{Prefix: "/remoton", TLSConfig: &tls.Config{
		InsecureSkipVerify: true,
	}}
	session := &remoton.SessionClient{Client: rclient,
		ID: "misessionid", AuthToken: "mitoken",
		APIURL: "https://miserver.com:9934"}
	
	//now you can dial any service
	conn, err := session.Dial("chat")
	//use conn -net.Conn-
~~~
