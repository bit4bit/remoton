package main

import (
	"net"
	"net/http"
	"net/rpc"

	"github.com/bit4bit/remoton"
)

func testCounter() {

	rclient := remoton.Client{Prefix: "/remoton"}
	session, err := rclient.NewSession("http://localhost:3000", "public")
	if err != nil {
		panic(err)
	}

	defer session.Destroy()
	listener := session.Listen("counter")
	println("Listen Counter")
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		println("\tNew connection to counter")
		go func(conn net.Conn) {
			var counter byte

			for {
				_, err := conn.Write([]byte{counter})
				if err != nil {
					println("\tClose connection")
					break
				}

				counter += 1
			}

		}(conn)

	}
}

type Args struct {
	A, B int
}

type Arith int

func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

func testRpc() {
	rclient := remoton.Client{Prefix: "/remoton"}
	session, err := rclient.NewSession("http://localhost:3000", "public")
	if err != nil {
		panic(err)
	}

	defer session.Destroy()
	listener := session.Listen("rpc")
	println("Listen RPC")
	arith := new(Arith)
	srv := rpc.NewServer()
	srv.Register(arith)
	srv.Accept(listener)
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("./test/")))
	http.Handle("/remoton/", http.StripPrefix("/remoton",
		remoton.NewServer(func(auth string, r *http.Request) bool {
			return auth == "public"
		}, func() string {
			return "test"
		})),
	)
	go testCounter()
	go testRpc()
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		panic(err)
	}
}
