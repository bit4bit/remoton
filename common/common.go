package common

import (
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"strconv"
)

var (
	ErrRootCertifcate = errors.New("invalid root certificate")
)

func GetServerInsecure(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return ""
	}
	iport, _ := strconv.Atoi(port)
	return net.JoinHostPort(host, strconv.Itoa(iport-1))
}

func GetRootCAFromFile(file string) (*x509.CertPool, error) {
	roots := x509.NewCertPool()
	rootPEM, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	ok := roots.AppendCertsFromPEM([]byte(rootPEM))
	if !ok {
		return nil, ErrRootCertifcate
	}
	return roots, nil
}

func FindFreePortTCP(startPort int) (string, int) {

	for ; startPort < 65534; startPort++ {
		conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(startPort))
		if err != nil {
			return strconv.Itoa(startPort), startPort
		}
		conn.Close()
	}
	panic("cant find free port")
}
