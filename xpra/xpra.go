package xpra

import (
	"bytes"
	"errors"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

var (
	//ErrNotXPRA not found on system xpra
	ErrNotXPRA = errors.New("Failed not found executable xpra")

	//ErrClosingTCP error xpra
	ErrClosingTCP = errors.New("closing tcp socket")
)

var (
	xpraPath    string
	xpraPathErr error
)

var (
	xpraArgsAttach = []string{
		"attach",
	}

	xpraArgsBind = []string{
		"shadow", ":0", "--mdns=no",
	}
)

type Xpraer interface {
	SetPassword(password string)
	Attach(addr string) error
	Bind(addr string) error
	Version() string
	Terminate()
}

type Xpra struct {
	password     string
	passwordFile string
	addrAttach   string
	addrBind     string
}

func (c *Xpra) SetPassword(pass string) {
	c.password = pass
	c.passwordFile = generaPasswdFile(pass)
}

//Version of system xpra
func (c *Xpra) Version() string {
	if xpraPathErr != nil {
		return ""
	}

	xpraCmd := exec.Command(xpraPath, "--version")
	out, err := xpraCmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Split(string(out), " ")[1])
}

//Attach to xpra
func (c *Xpra) Attach(addr string) error {

	if xpraPathErr != nil {
		log.Error("xpra_attach:", xpraPathErr)
		return xpraPathErr
	}

	c.addrAttach = addr
	args := append(xpraArgsAttach, "tcp:"+addr)
	args = append(args, "--min-speed=30", "--min-quality=50",
		"--windows=yes",
		"--notifications=no", "--speaker=off", "--auto-refresh-delay=0.8",
		"--scaling=80")
	if c.passwordFile != "" {
		args = append(args, "--auth=file", "--password-file="+c.passwordFile)
	}
	args = platformAttachArgs(args)

	xpraCmd := exec.Command(xpraPath, args...)
	platformCmd(xpraCmd)
	if err := xpraCmd.Start(); err != nil {
		log.Error("xpra_attach:", err)
		return err
	}

	time.Sleep(time.Second)
	return nil
}

//Bind a xpra for listen connections
func (c *Xpra) Bind(addr string) error {
	if xpraPathErr != nil {
		log.Error("xpra_attach:", xpraPathErr)
		return xpraPathErr
	}

	var out bytes.Buffer
	args := append(xpraArgsBind, "--bind-tcp="+addr)
	if c.passwordFile != "" {
		args = append(args, "--auth=file", "--password-file="+c.passwordFile)
	}
	c.addrBind = addr
	args = platformBindArgs(args)
	log.Println("XpraBind args: ", args)
	xpraCmd := exec.Command(xpraPath, args...)
	platformCmd(xpraCmd)

	xpraCmd.Stderr = &out
	xpraCmd.Stdout = &out
	if err := xpraCmd.Start(); err != nil {
		log.Error("xpra_bind:", err)
		return err
	}

	xprayReady := regexp.MustCompile(`xpra is ready.`)
	xprayError := regexp.MustCompile("failed")
	xprayClosing := regexp.MustCompile("closing tcp socket localhost")
	for {
		time.Sleep(time.Second)
		log.Println(out.String())
		if xprayReady.Match(out.Bytes()) {
			return nil
		}

		if xprayError.Match(out.Bytes()) {
			str, _ := out.ReadString('\n')
			return errors.New(str)
		}
		if xprayClosing.Match(out.Bytes()) {
			log.Error(out.String())
			exec.Command("pkill", "xpra").Output()
			return ErrClosingTCP
		}
	}
}

//Terminate running xpra
func (c *Xpra) Terminate() {
	if xpraPathErr == nil {
		args := []string{"stop"}
		if c.addrAttach != "" {
			args = append(args, "tcp:"+c.addrAttach)
		} else if c.addrBind != "" {
			args = append(args, "tcp:"+c.addrBind)
		}

		if c.passwordFile != "" {
			args = append(args, "--password-file="+c.passwordFile)
		}
		exec.Command(xpraPath, args...).Start()
		syscall.Unlink(c.passwordFile)
	}
}

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

	return passwdFile.Name()
}
