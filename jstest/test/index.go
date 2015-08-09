//go:generate gopherjs build -m index.go

package main

import (
	"fmt"
	"net/rpc"

	"github.com/bit4bit/remoton"
	"github.com/rusco/qunit"
)

type Args struct {
	A, B int
}

func main() {
	rclient := &remoton.Client{Prefix: "/remoton"}

	qunit.Module("remoton")
	qunit.AsyncTest("RPC", func() interface{} {
		qunit.Expect(1)
		go func() {
			defer qunit.Start()
			session := &remoton.SessionClient{Client: rclient,
				ID: "test", APIURL: "http://localhost:3000"}

			conn, err := session.Dial("rpc")
			if err != nil {
				qunit.Ok(false, err.Error())
			}
			defer conn.Close()

			rpcclient := rpc.NewClient(conn)
			fmt.Println("remote call")
			var reply int
			err = rpcclient.Call("Arith.Multiply", &Args{7, 5}, &reply)
			if err != nil {
				qunit.Ok(false, "Call remote function:"+err.Error())
				return
			}
			defer rpcclient.Close()
			if reply == 35 {
				qunit.Ok(true, "Multiply 7 * 5 == 35")
			} else {
				qunit.Ok(false, "Invalid reply")
			}
		}()
		return nil
	})
	qunit.AsyncTest("Stream", func() interface{} {
		qunit.Expect(1)
		go func() {
			defer qunit.Start()
			session := &remoton.SessionClient{Client: rclient,
				ID: "test", APIURL: "http://localhost:3000"}

			conn, err := session.Dial("counter")
			if err != nil {
				qunit.Ok(false, err.Error())
			}
			defer conn.Close()
			var counter byte
			for {
				buf := make([]byte, 1)
				_, err := conn.Read(buf)
				if err != nil {
					panic(err)
				}
				rcounter := buf[0]
				if rcounter != counter {
					qunit.Ok(false,
						fmt.Sprintf("Unexpected counter was %d expect %d", rcounter, counter))
					return
				}
				fmt.Println(counter, rcounter)
				counter += 1
				if counter == 10 {
					break
				}
			}
			qunit.Ok(true, "Stream verified")

		}()
		return nil
	})

}
