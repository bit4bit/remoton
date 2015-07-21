package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"

	log "github.com/Sirupsen/logrus"
	"github.com/bit4bit/remoton"
)

type Session struct {
	ID        string
	AuthToken string
}

func (c Session) String() string {
	return fmt.Sprintf("Session[%s]=%s \n\t%s:%s",
		c.ID, c.AuthToken,
		c.ID, c.AuthToken)
}

var (
	srv        = flag.String("srv", "localhost:9934", "server address")
	service    = flag.String("service", "chat", "service chat")
	tunnelAddr = flag.String("tunnel", "localhost:9990", "tunnel port")
	authToken  = flag.String("auth", "", "auth token")
	srvPrefix  = flag.String("srv-prefix", "/remoton", "base app default remoton")
)

func main() {
	flag.Parse()

	if *authToken == "" {
		log.Error("need auth token please use -auth")
		return
	}

	rclient := remoton.Client{Prefix: *srvPrefix, TLSConfig: &tls.Config{
		InsecureSkipVerify: true,
	}}
	session, err := rclient.NewSession("https://"+*srv, *authToken)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Session -> %s:%s", session.ID, session.AuthToken)
	defer session.Destroy()

	wsconn, err := session.Dial(*service)
	if err != nil {
		log.Fatal(err)
	}

	listen, err := net.Listen("tcp", *tunnelAddr)
	if err != nil {
		log.Fatal(err)
	}

	for {
		log.Println("listen at ", *tunnelAddr)
		conn, err := listen.Accept()
		if err != nil {
			log.Fatal(err)
		}

		errc := make(chan error, 2)
		cp := func(dst net.Conn, src net.Conn) {
			_, err := io.Copy(dst, src)
			errc <- err
		}

		go cp(conn, wsconn)
		go cp(wsconn, conn)

		log.Error(<-errc)
		conn.Close()
		wsconn.Close()
		wsconn, err = session.Dial(*service)
		if err != nil {
			log.Error(err)
			return
		}
	}
}
