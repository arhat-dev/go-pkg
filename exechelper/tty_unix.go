// +build !windows,!js,!plan9,!aix

package exechelper

import (
	"io"
	"os/exec"

	"github.com/creack/pty"

	"arhat.dev/pkg/log"
)

func startCmdWithTty(logger log.Interface, cmd *exec.Cmd, stdin io.Reader, stdout io.Writer, resizeH ResizeHandleFunc) (func(), error) {
	f, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	if stdin != nil {
		go func() {
			_, err := io.Copy(f, stdin)
			if err != nil {
				logger.D("exception in writing stdin data", log.Error(err))
			}
		}()
	}

	if stdout != nil {
		go func() {
			_, err := io.Copy(stdout, f)
			if err != nil {
				logger.D("exception in reading stdout data", log.Error(err))
			}
		}()
	}

	go func() {
		doResize := func(cols, rows uint64) error {
			return pty.Setsize(f, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
		}

		for resizeH(doResize) {
		}
	}()

	return func() { _ = f.Close() }, nil
}
