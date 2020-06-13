package exechelper

import (
	"arhat.dev/pkg/log"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"arhat.dev/pkg/wellknowerrors"
)

const (
	DefaultExitCodeOnError = 128
)

type ResizeHandleFunc func(doResize func(cols, rows uint64) error) (more bool)

func DoHeadless(command []string, env map[string]string) (int, error) {
	return Do(nil, nil, nil, nil, command, false, env)
}

func Prepare(command []string, tty bool, env map[string]string) *exec.Cmd {
	cmd := exec.Command(command[0], command[1:]...)
	// if using tty in unix, github.com/creack/pty will Setsid, and if we
	// Setpgid, will fail the process creation
	cmd.SysProcAttr = getSysProcAttr(tty)

	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	return cmd
}

// Do execute command directly in host
func Do(logger log.Interface, stdin io.Reader, stdout, stderr io.Writer, resizeH ResizeHandleFunc, command []string, tty bool, env map[string]string) (int, error) {
	if len(command) == 0 {
		// impossible for agent exec, but still check for test
		return DefaultExitCodeOnError, fmt.Errorf("empty command: %w", wellknownerrors.ErrInvalidOperation)
	}

	cmd := Prepare(command, tty, env)
	if tty {
		cleanup, err := startCmdWithTty(logger, cmd, stdin, stdout, resizeH)
		if err != nil {
			return DefaultExitCodeOnError, err
		}
		defer cleanup()
	} else {
		cmd.Stdin = stdin
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		if err := cmd.Start(); err != nil {
			return DefaultExitCodeOnError, err
		}
	}

	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode(), err
		}

		if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrClosedPipe) {
			return DefaultExitCodeOnError, err
		}
	}

	return 0, nil
}
