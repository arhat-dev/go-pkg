// +build windows js plan9 aix

package exechelper

import (
	"io"
	"os/exec"

	"arhat.dev/pkg/wellknowerrors"
)

func startCmdWithTty(logger log.Interface, cmd *exec.Cmd, stdin io.Reader, stdout io.Writer, resizeH ResizeHandleFunc) (func(), error) {
	return nil, wellknownerrors.ErrNotSupported
}
