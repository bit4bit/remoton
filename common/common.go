package common

import (
	"crypto/x509"
	"errors"
	"io/ioutil"
)

var (
	ErrRootCertifcate = errors.New("invalid root certificate")
)

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
