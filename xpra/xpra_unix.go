/*xpra interface to command xpra
 */
package xpra

import (
	"bytes"
	"errors"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"regexp"
	"strconv"
	"syscall"
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
	xpraCmd *exec.Cmd
)

func Attach(addr string) error {
	pid := load_pid(PID_FILE_SUPPORT)
	log.Println(pid)
	if pid > 0 {
		syscall.Kill(pid, syscall.SIGKILL)
		syscall.Unlink(get_pid_path(PID_FILE_SUPPORT))
	}

	xpraPath, err := exec.LookPath("xpra")
	if err != nil {
		log.Error("xpra_attach:", err)
		return err
	}

	xpraCmd = exec.Command(xpraPath, "attach", "tcp:"+addr)

	if err := xpraCmd.Start(); err != nil {
		log.Error("xpra_attach:", err)
		return err
	}
	time.Sleep(time.Second)
	return nil
}

func Bind(addr string) error {
	pid := load_pid(PID_FILE_CLIENT)
	log.Println(pid)
	if pid > 0 {
		syscall.Kill(pid, syscall.SIGKILL)
		syscall.Unlink(get_pid_path(PID_FILE_CLIENT))
	}

	xpraPath, err := exec.LookPath("xpra")
	if err != nil {
		log.Error("xpra_bind:", err)
		return err
	}

	var out bytes.Buffer

	xpraCmd = exec.Command(xpraPath, "shadow", ":0", "--no-daemon", "--no-mdns", "--bind-tcp="+addr)
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
			save_pid(xpraCmd.Process.Pid, PID_FILE_CLIENT)
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

	return nil
}

func Terminate() {
	if xpraCmd != nil && xpraCmd.Process != nil {
		xpraCmd.Process.Kill()
	}
}

func get_pid_path(pid_name string) string {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	if pid_name == "" {
		panic("invalid pid_name can't be empty")
	}

	return path.Join(u.HomeDir, pid_name)
}
func save_pid(pid int, pid_name string) {

	pidPath := get_pid_path(pid_name)
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {

		err := ioutil.WriteFile(pidPath, []byte(strconv.Itoa(pid)), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
}

func load_pid(pid_name string) int {

	pidPath := get_pid_path(pid_name)
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
