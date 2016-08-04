package main

import (
	"flag"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"

	"github.com/bit4bit/remoton"
	"github.com/satori/go.uuid"
	
	log "github.com/Sirupsen/logrus"

	"github.com/throttled/throttled"
	"github.com/throttled/throttled/store"
)

var (
	listenAddr    = flag.String("listen", "localhost:9934", "listen address")
	authTokenFlag = flag.String("auth-token", "", "authenticate API")
	certFile      = flag.String("cert", "cert.pem", "cert pem")
	keyFile       = flag.String("key", "key.pem", "key pem")
	profile       = flag.String("cpuprofile", "", "output profile to file")
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

	if *authTokenFlag == "" {
		*authTokenFlag = "public"
		log.Println("Using default Token", *authTokenFlag)
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
		remoton.NewServer(func(authToken string, r *http.Request) bool {
			return authToken == *authTokenFlag
		}, func() string {
			return uuid.NewV4().String()[0:8]
		})))

	log.Println("Listen at HTTPS ", *listenAddr)
	sSecure := &http.Server{
		Addr:    *listenAddr,
		Handler: th.Throttle(mux),
	}
	host, port, err := net.SplitHostPort(*listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	iport, err := strconv.Atoi(port)
	if err != nil {
		panic(err)
	}

	listenInsecureAddr := net.JoinHostPort(host, strconv.Itoa(iport-1))
	//Default insecure it's a previous port
	log.Println("Listen at HTTP ", listenInsecureAddr)
	sInsecure := &http.Server{
		Addr:    listenInsecureAddr,
		Handler: th.Throttle(mux),
	}
	go sInsecure.ListenAndServe()

	log.Fatal(sSecure.ListenAndServeTLS(*certFile, *keyFile))
}
