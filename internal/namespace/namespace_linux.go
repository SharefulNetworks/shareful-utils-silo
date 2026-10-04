//go:build linux

// Package namespace contains the Linux namespace and shell-launching logic.
package namespace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/SharefulNetworks/shareful-utils-emptyshell/internal/session"
	"golang.org/x/sys/unix"
)

// EnterMountNamespace creates a new mount namespace and makes the namespace's
// mount changes private.
//
// This is important because mounting OverlayFS must not modify the host's
// mount namespace. The child process gets its own view of the mount table.
//
// The current prototype uses the root user's existing privileges rather than
// an unprivileged user namespace.
func EnterMountNamespace(s *session.Session) error {
	if err := unix.Unshare(unix.CLONE_NEWNS); err != nil {
		return fmt.Errorf("create mount namespace: %w", err)
	}

	// Prevent mount/unmount operations in this namespace from propagating back
	// to the host's mount namespace.
	if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("make mounts private: %w", err)
	}

	// The overlay is mounted after entering the private mount namespace.
	// This means the mount exists only in this process's namespace.
	return nil
}

// RunShell starts an interactive Bash shell with the overlay as its root.
func RunShell(s *session.Session) error {
	promptName := s.Name
	if s.Transient {
		if promptName == "" {
			promptName = filepath.Base(s.Root)
		}
	} else if promptName == "" {
		promptName = "unnamed"
	}

	bootstrap := fmt.Sprintf(
		"export PS1=%s; mount -t proc proc /proc && exec /bin/bash --noprofile --norc -i",
		shellQuote("\\[\\e[92m\\]silo#"+promptName+"> \\[\\e[0m\\] "),
	)
	cmd := exec.Command("/bin/bash", "-c", bootstrap)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Chroot the child into the overlay so it sees the mounted filesystem as
	// its root rather than as a subdirectory of the host system.
	// Addionally, we also enter a new PID namespace so that the child process sees itself as PID 1
	// and can manage its own processes independently of the host system. (i.e. process isolation)
	cmd.Dir = "/"
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:     s.Merged,
		Cloneflags: syscall.CLONE_NEWPID,
	}

	env := make([]string, 0, len(os.Environ())+3)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "PROMPT_COMMAND=") {
			continue
		}
		env = append(env, entry)
	}
	cmd.Env = append(env,
		"EMPTY_SHELL=1",
		"EMPTY_SHELL_ROOT=/",
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell: %w", err)
	}

	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
