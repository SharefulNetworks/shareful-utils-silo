//go:build linux

// Package namespace contains the Linux namespace and shell-launching logic.
package namespace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	cmd := exec.Command("/bin/bash", "--noprofile", "--norc")

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Chroot the child into the overlay so it sees the mounted filesystem as
	// its root rather than as a subdirectory of the host system.
	cmd.Dir = "/"
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot: s.Merged,
	}

	promptName := s.Name
	if s.Transient {
		if promptName == "" {
			promptName = filepath.Base(s.Root)
		} else {
			promptName = promptName
		}
	} else if promptName == "" {
		promptName = "unnamed"
	}
	cmd.Env = append(os.Environ(),
		"EMPTY_SHELL=1",
		"EMPTY_SHELL_ROOT=/",
		"PS1=silo#"+promptName+"> ",
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell: %w", err)
	}

	return nil
}
