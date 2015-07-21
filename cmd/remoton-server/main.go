package main

import (
	"flag"
	"net/http"
	"os"
	"runtime/pprof"

	"github.com/bit4bit/remoton"

	"code.google.com/p/go-uuid/uuid"
	"github.com/PuerkitoBio/throttled"
	"github.com/PuerkitoBio/throttled/store"
	log "github.com/Sirupsen/logrus"
)

var (
	listenAddr = flag.String("listen", "localhost:9934", "listen address")
	authToken  = flag.String("auth-token", "", "authenticate API")
	certFile   = flag.String("cert", "cert.pem", "cert pem")
	keyFile    = flag.String("key", "key.pem", "key pem")
	profile    = flag.String("cpuprofile", "", "output profile to file")
)

func main() {
	flag.Parse()

	if *profile != "" {
		flag, err := os.Create(*profile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(flag)
		defer pprof.StopCPUProfile()
	}

	if *authToken == "" {
		*authToken = "public"
		log.Println("Using default Token", *authToken)
	}

	if *certFile == "" || *keyFile == "" {
		log.Error("need cert file and key file .pem")
		return
	}
	th := throttled.RateLimit(throttled.PerMin(30),
		&throttled.VaryBy{RemoteAddr: true},
		store.NewMemStore(100),
	)
	srv := remoton.NewServer(*authToken, func() (string, string) {
		return uuid.New()[0:8], uuid.New()[0:6]
	})
	log.Println("Listen at HTTPS ", *listenAddr)
	log.Fatal(http.ListenAndServeTLS(*listenAddr, *certFile, *keyFile, th.Throttle(srv)))
}
