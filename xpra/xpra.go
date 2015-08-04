package xpra

import (
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

	passwdFile.Write([]byte(password))
	tmpFiles = append(tmpFiles, passwdFile.Name())
	return passwdFile.Name()
}

func cleanTempFiles() {
	for _, tmpf := range tmpFiles {
		os.Remove(tmpf)
	}
}
