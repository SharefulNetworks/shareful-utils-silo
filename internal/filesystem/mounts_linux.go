//go:build linux

package filesystem

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// MountSpec describes a filesystem mount that should exist inside the sandbox.
type MountSpec struct {
	Source string
	Target string
	Type   string
	Flags  uintptr
	Data   string
}

// MountHandler manages the runtime mounts required by language toolchains.
type MountHandler struct {
	mounts []MountSpec
}

// NewDefaultMountHandler returns the default mount set used by a developer-friendly Silo.
func NewDefaultMountHandler() *MountHandler {
	return &MountHandler{
		mounts: []MountSpec{
			{Source: "proc", Target: "/proc", Type: "proc", Flags: 0},
			{Source: "tmpfs", Target: "/tmp", Type: "tmpfs", Flags: unix.MS_NOSUID | unix.MS_NODEV},
			{Source: "tmpfs", Target: "/run", Type: "tmpfs", Flags: unix.MS_NOSUID | unix.MS_NODEV},
			{Source: "tmpfs", Target: "/dev", Type: "devtmpfs", Flags: unix.MS_NOSUID},
			{Source: "devpts", Target: "/dev/pts", Type: "devpts", Flags: unix.MS_NOSUID | unix.MS_NOEXEC},
			{Source: "sysfs", Target: "/sys", Type: "sysfs", Flags: unix.MS_NOSUID | unix.MS_NOEXEC | unix.MS_RDONLY},
		},
	}
}

// Apply mounts all default runtime filesystems into the chroot root.
func (m *MountHandler) Apply(root string) error {
	for _, spec := range m.mounts {

		// NB: we now skip mounting /proc here because in order for process isolation
		//     to work correctly, /proc must be mounted inside the chroot after
		//     entering the new PID namespace. This is done in RunShell in
		//     namespace_linux.go.
		if spec.Target == "/proc" {
			continue
		}

		target := filepath.Join(root, spec.Target)
		if err := os.MkdirAll(target, 0755); err != nil {
			return fmt.Errorf("create mount target %s: %w", target, err)
		}

		if err := unix.Mount(spec.Source, target, spec.Type, spec.Flags, spec.Data); err != nil {
			return fmt.Errorf("mount %s -> %s: %w", spec.Source, target, err)
		}
	}

	for _, pair := range []struct {
		src string
		dst string
	}{
		{src: "/etc/resolv.conf", dst: filepath.Join(root, "etc", "resolv.conf")},
		{src: "/etc/hosts", dst: filepath.Join(root, "etc", "hosts")},
		{src: "/etc/ssl/certs", dst: filepath.Join(root, "etc", "ssl", "certs")},
	} {
		if err := bindIfExists(pair.src, pair.dst); err != nil {
			return err
		}
	}

	return nil
}

// Cleanup unmounts the runtime mounts in the reverse order they were attached.
func (m *MountHandler) Cleanup(root string) {
	for i := len(m.mounts) - 1; i >= 0; i-- {
		target := filepath.Join(root, m.mounts[i].Target)
		unmountIfMounted(target)
	}

	for _, dst := range []string{
		filepath.Join(root, "etc", "resolv.conf"),
		filepath.Join(root, "etc", "hosts"),
		filepath.Join(root, "etc", "ssl", "certs"),
	} {
		unmountIfMounted(dst)
	}
}

func unmountIfMounted(target string) {
	if _, err := os.Stat(target); err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	if err := unix.Unmount(target, unix.MNT_DETACH); err != nil {
		_ = err
	}
}

// bindIfExists bind mounts src onto dst.
//
// The target is created to match the type of the source:
//   - directories: the target directory is created with MkdirAll
//   - files: the target file is created before the bind mount
//
// This is important when using a minimal Silo root filesystem because files
// such as /etc/hosts may not exist in the packaged image.
func bindIfExists(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", src, err)
	}

	if info.IsDir() {
		if err := os.MkdirAll(dst, 0755); err != nil {
			return fmt.Errorf("create bind target %s: %w", dst, err)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf(
				"create bind target directory %s: %w",
				filepath.Dir(dst),
				err,
			)
		}

		// A file bind mount requires the destination file to exist.
		f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return fmt.Errorf("create bind target %s: %w", dst, err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("close bind target %s: %w", dst, err)
		}
	}

	if err := unix.Mount(src, dst, "", unix.MS_BIND, ""); err != nil {
		return fmt.Errorf("bind mount %s -> %s: %w", src, dst, err)
	}

	return nil
}
