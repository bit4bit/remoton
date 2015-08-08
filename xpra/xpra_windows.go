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
}

func platformCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}

func platformAttachArgs(args []string) []string {
	return args
}

func platformBindArgs(args []string) []string {
	//BUG xpra on windows not work auth=file
	for idx, arg := range args {
		if arg == "--auth=file" {
			args[idx] = ""
		}
	}
	return args
}

func localIcon() string {
	return syscall.EscapeArg(filepath.Join(filepath.Dir(os.Args[0]), "icon.ico"))
}
