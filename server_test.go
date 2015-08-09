package remoton

import (
	"bufio"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

//TestDialAndListen test websocket tunnel
func TestDialAndListen(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/remoton/", http.StripPrefix("/remoton", NewServer(
		func(authToken string, r *http.Request) bool {
			return authToken == "testsrv"
		},
		func() string {
			return "testid"
		})))

	ts := httptest.NewTLSServer(mux)

	defer ts.Close()

	rclient := Client{Prefix: "/remoton", TLSConfig: &tls.Config{
		InsecureSkipVerify: true,
	}}

	session, err := rclient.NewSession(ts.URL, "testsrv")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Destroy()

	go func() {
		listener := session.Listen("test")

		lconn, err := listener.Accept()
		if err != nil {
			t.Error(err)
		}
		data, _ := bufio.NewReader(lconn).ReadString('\n')

		if data != "transfer" {
			t.Errorf("want %v get %v", "transfer", data)
		}
	}()

	dconn, err := session.Dial("test")
	dconn.Write([]byte("transfer"))
	dconn.Close()
}

//TestDialAndListenTCP tcp tunnel
func TestDialAndListenTCP(t *testing.T) {

	ts := httptest.NewTLSServer(NewServer(
		func(authToken string, r *http.Request) bool {
			return authToken == "testsrv"
		}, func() string {
			return "testid"
		}))

	defer ts.Close()

	rclient := Client{Prefix: "", TLSConfig: &tls.Config{
		InsecureSkipVerify: true,
	}}

	session, err := rclient.NewSession(ts.URL, "testsrv")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Destroy()

	go func() {
		listener := session.ListenTCP("test")

		lconn, err := listener.Accept()
		if err != nil {
			t.Error(err)
		}
		data, _ := bufio.NewReader(lconn).ReadString('\n')

		if data != "transfer" {
			t.Errorf("want %v get %v", "transfer", data)
		}
	}()

	dconn, err := session.DialTCP("test")
	dconn.Write([]byte("transfer"))
	dconn.Close()
}
