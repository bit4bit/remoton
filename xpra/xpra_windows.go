//Package xpra for windows
//+build windows
package xpra

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
)

func init() {
	xpraPath = path.Join(filepath.Dir(os.Args[0]), "Xpra", "xpra_cmd.exe")
	if _, err := os.Stat(xpraPath); err != nil && os.IsNotExist(err) {
		xpraPathErr = ErrNotXPRA
	}
	println(localIcon())
}

func platformCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}

func platformAttachArgs(args []string) []string {
	return append(args, "--window-icon="+localIcon(), "--tray-icon="+localIcon())
}

func platformBindArgs(args []string) []string {
	//BUG xpra on windows not work auth=file
	for idx, arg := range args {
		if arg == "--auth=file" {
			args = append(args[:idx], args[idx+1:]...)
			break
		}
	}
	return args
}

func localIcon() string {
	return filepath.Join(filepath.Dir(os.Args[0]), "icon.ico")
}
