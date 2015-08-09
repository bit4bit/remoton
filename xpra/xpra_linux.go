// +build linux

/*xpra interface to command xpra
 */

package xpra

import (
	"os/exec"
)

func init() {
	xpraPath, xpraPathErr = exec.LookPath("xpra")
}

func platformAttachArgs(args []string) []string {
	return args
}

func platformBindArgs(args []string) []string {
	return append(args, "--daemon=no", "--notifications=no", "--speaker=off")
}

func platformCmd(cmd *exec.Cmd) {

}
