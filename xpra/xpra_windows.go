//Package xpra for windows
//+build windows
package xpra

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

var (
	//ErrNotXPRA can't find xpra on system
	ErrNotXPRA = errors.New("Failed not found executable xpra")

	//ErrClosingTCP a error from xpra
	ErrClosingTCP = errors.New("closing tcp socket")
)

var (
	xpraCmd     *exec.Cmd
	pathXpraCmd = path.Join(pathProgramFiles(), "Xpra", "xpra_cmd")
)

//Version of xpra
func Version() string {
	xpraCmd = exec.Command(pathXpraCmd, "--version")
	out, err := xpraCmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(strings.Split(string(out), " ")[1])
}

//Attach to xpra server
func Attach(addr, password string) error {
	xpraCmd = exec.Command(pathXpraCmd, "attach", "tcp:"+addr, "--auth=file",
		"--password-file="+generaPasswdFile(password))

	if err := xpraCmd.Start(); err != nil {
		log.Error("xpra_attach:", err)
		return err
	}
	time.Sleep(time.Second)
	return nil
}

//Bind a xpra server for listen connections
func Bind(addr, password string) error {
	var out bytes.Buffer

	xpraCmd = exec.Command(pathXpraCmd, "shadow", ":0", "--no-mdns",
		"--bind-tcp="+addr, "--auth=file",
		"--password-file="+generaPasswdFile(password))
	xpraCmd.Stderr = &out
	if err := xpraCmd.Start(); err != nil {
		log.Error("xpra_bind:", err)
		return err
	}
	xprayReady := regexp.MustCompile(`xpra is ready.`)
	xprayError := regexp.MustCompile("failed|error")
	xprayClosing := regexp.MustCompile("closing tcp socket localhost")
	errc := make(chan error)
	stopWait := make(chan struct{})
	go func(out *bytes.Buffer) {
		for {
			select {
			case <-time.After(time.Second):
				log.Debugln("waiting action xpra")
				log.Println(out.String())
				if xprayReady.Match(out.Bytes()) {
					errc <- nil
					break
				}

				if xprayError.Match(out.Bytes()) {
					str, _ := out.ReadString('\n')
					errc <- errors.New(str)
					break
				}
				if xprayClosing.Match(out.Bytes()) {
					log.Error(out.String())
					errc <- ErrClosingTCP
					break
				}
			case <-stopWait:
				break
			}
		}
	}(&out)

	select {
	case <-time.After(time.Second * 10):

	case err := <-errc:
		return err
	}

	stopWait <- struct{}{}
	return errors.New("Failed start xpra server")
}

//Terminate the running xpra
func Terminate() {
	if xpraCmd != nil && xpraCmd.Process != nil {
		xpraCmd.Process.Kill()
	}
	cleanTempFiles()
}

func init() {
	log.Println(pathProgramFiles())
}

func pathProgramFiles() string {
	if runtime.GOARCH == "amd64" {
		return os.Getenv("ProgramFiles(x86)")
	}
	return os.Getenv("ProgramFiles")

}
