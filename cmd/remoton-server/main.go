package main

import (
	"flag"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/bit4bit/remoton"

	"code.google.com/p/go-uuid/uuid"
	log "github.com/Sirupsen/logrus"

	"github.com/throttled/throttled"
	"github.com/throttled/throttled/store"
)

var (
	listenAddr         = flag.String("listen", "localhost:9934", "listen address")
	listenAddrInsecure = flag.String("listen-insecure", "localhost:9933",
		"listen addres insecure")
	authToken = flag.String("auth-token", "", "authenticate API")
	certFile  = flag.String("cert", "cert.pem", "cert pem")
	keyFile   = flag.String("key", "key.pem", "key pem")
	profile   = flag.String("cpuprofile", "", "output profile to file")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

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
	mux := http.NewServeMux()
	mux.Handle("/remoton/", http.StripPrefix("/remoton",
		remoton.NewServer(*authToken, func() string {
			return uuid.New()[0:8]
		})))

	log.Println("Listen at HTTPS ", *listenAddr)
	sSecure := &http.Server{
		Addr:    *listenAddr,
		Handler: th.Throttle(mux),
	}

	sInsecure := &http.Server{
		Addr:    *listenAddrInsecure,
		Handler: th.Throttle(mux),
	}
	go sInsecure.ListenAndServe()

	log.Fatal(sSecure.ListenAndServeTLS(*certFile, *keyFile))
}
