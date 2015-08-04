package xpra

import (
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
)

var (
	tmpFiles = make([]string, 0)
)

func generaPasswdFile(password string) string {
	passwdFile, err := ioutil.TempFile(
		os.TempDir(), "passwdxpraremoton",
	)

	if err != nil {
		panic(err)
	}

	log.Println(passwdFile.Name())
	passwdFile.Write([]byte(password))
	passwdFile.Close()

	tmpFiles = append(tmpFiles, passwdFile.Name())
	return passwdFile.Name()
}

func cleanTempFiles() {
	for _, tmpf := range tmpFiles {
		os.Remove(tmpf)
	}
}
