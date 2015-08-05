// +build linux

/*xpra interface to command xpra
 */

package xpra

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	pidFileClient  = ".remotonclient.pid"
	pidFileSupport = ".remotonsupport.pid"
)

var (
	//ErrNotXPRA not found on system xpra
	ErrNotXPRA = errors.New("Failed not found executable xpra")

	//ErrClosingTCP error xpra
	ErrClosingTCP = errors.New("closing tcp socket")
)

var (
	xpraCmd *exec.Cmd
)

//Version of system xpra
func Version() string {
	xpraPath, err := exec.LookPath("xpra")
	if err != nil {
		return ""
	}

	xpraCmd = exec.Command(xpraPath, "--version")
	out, err := xpraCmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Split(string(out), " ")[1])
}

//Attach to xpra
func Attach(addr, password string) error {
	pid := loadPid(pidFileSupport)
	log.Println(pid)
	if pid > 0 {
		syscall.Kill(pid, syscall.SIGKILL)
		syscall.Unlink(getPidPath(pidFileSupport))
	}

	xpraPath, err := exec.LookPath("xpra")
	if err != nil {
		log.Error("xpra_attach:", err)
		return err
	}

	xpraCmd = exec.Command(xpraPath, "attach", "tcp:"+addr, "-z9",
		"--password-file="+generaPasswdFile(password), "--auth=file")

	if err := xpraCmd.Start(); err != nil {
		log.Error("xpra_attach:", err)
		return err
	}
	time.Sleep(time.Second)
	return nil
}

//Bind a xpra for listen connections
func Bind(addr, password string) error {
	pid := loadPid(pidFileClient)
	log.Println(pid)
	if pid > 0 {
		syscall.Kill(pid, syscall.SIGKILL)
		syscall.Unlink(getPidPath(pidFileClient))
	}

	xpraPath, err := exec.LookPath("xpra")
	if err != nil {
		log.Error("xpra_bind:", err)
		return err
	}

	var out bytes.Buffer

	xpraCmd = exec.Command(xpraPath, "shadow", ":0",
		"--daemon=no", "--mdns=no",
		"--bind-tcp="+addr, "--auth=file", "--password-file="+generaPasswdFile(password))
	xpraCmd.Stderr = &out

	if err := xpraCmd.Start(); err != nil {
		log.Error("xpra_bind:", err)
		return err
	}

	xprayReady := regexp.MustCompile(`xpra is ready.`)
	xprayError := regexp.MustCompile("failed")
	xprayClosing := regexp.MustCompile("closing tcp socket localhost")
	for {
		time.Sleep(time.Second)

		if xprayReady.Match(out.Bytes()) {
			savePid(xpraCmd.Process.Pid, pidFileClient)
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
func Terminate() {
	if xpraCmd != nil && xpraCmd.Process != nil {
		xpraCmd.Process.Kill()
	}
	cleanTempFiles()
}

func getPidPath(pidName string) string {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	if pidName == "" {
		panic("invalid pid_name can't be empty")
	}

	return path.Join(u.HomeDir, pidName)
}

func savePid(pid int, pidName string) {

	pidPath := getPidPath(pidName)
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {

		err := ioutil.WriteFile(pidPath, []byte(strconv.Itoa(pid)), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
}

func loadPid(pidName string) int {

	pidPath := getPidPath(pidName)
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {

	} else {
		spid, err := ioutil.ReadFile(pidPath)
		if err != nil {
			panic(err)
		}
		pid, err := strconv.Atoi(string(spid))
		if err != nil {
			panic(err)
		}

		return pid
	}

	return -1
}
