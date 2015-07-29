//+build windows
package xpra

import (
	"bytes"
	"errors"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"path"
	"regexp"
	"time"
)

const (
	PID_FILE_CLIENT  = ".remotonclient.pid"
	PID_FILE_SUPPORT = ".remotonsupport.pid"
)

var (
	ErrNotXPRA    = errors.New("Failed not found executable xpra")
	ErrClosingTCP = errors.New("closing tcp socket")
)

var (
	xpraCmd     *exec.Cmd
	pathXpraCmd = path.Join(pathProgramFiles(), "Xpra", "xpra_cmd")
)

func Attach(addr string) error {
	xpraCmd = exec.Command(pathXpraCmd, "attach", "tcp:"+addr)

	if err := xpraCmd.Start(); err != nil {
		log.Error("xpra_attach:", err)
		return err
	}
	time.Sleep(time.Second)
	return nil
}

func Bind(addr string) error {
	var out bytes.Buffer

	xpraCmd = exec.Command(pathXpraCmd, "shadow", ":0", "--no-mdns", "--bind-tcp="+addr)
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

func Terminate() {
	if xpraCmd != nil && xpraCmd.Process != nil {
		xpraCmd.Process.Kill()
	}
}

func init() {
	log.Println(pathProgramFiles())
}

func pathProgramFiles() string {
	return os.Getenv("ProgramFiles")
}
