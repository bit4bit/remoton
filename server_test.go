package remoton

import (
	"bufio"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

//TestDialAndListen test websocket tunnel
func TestDialAndListen(t *testing.T) {
	maxConns := 10
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

		for i := 0; i < maxConns; i++ {
			lconn, err := listener.Accept()
			if err != nil {
				t.Error(err)
			}
			data, _ := bufio.NewReader(lconn).ReadString('\n')
			if strings.TrimSpace(data) != "transfer" {
				t.Errorf("want %v get %v", "transfer", data)
			} else {
				lconn.Write([]byte("feedback"))
				lconn.Write([]byte{'\n'})
			}
		}
	}()

	wg := &sync.WaitGroup{}
	worker := func() {
		go func() {
			dconn, err := session.Dial("test")
			if err != nil {
				t.Fatal(err)
			}
			dconn.Write([]byte("transfer"))
			dconn.Write([]byte{'\n'})
			data, err := bufio.NewReader(dconn).ReadString('\n')
			if err != nil {
				t.Error(err)
			}
			if strings.TrimSpace(data) != "feedback" {
				t.Errorf("want %v get %v", "feedback", data)
			}
			dconn.Close()
			wg.Done()
		}()
	}
	for i := 0; i < maxConns; i++ {
		wg.Add(1)
		worker()
	}
	wg.Wait()
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
