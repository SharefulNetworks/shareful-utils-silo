//go:build linux

// Package filesystem contains the Linux filesystem isolation used by Silo.
package filesystem

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SharefulNetworks/shareful-utils-silo/internal/session"
	"golang.org/x/sys/unix"
)

// SetupOverlay mounts an OverlayFS over the host root filesystem.
// The host filesystem becomes the read-only lower layer. Any writes made
// inside Silo are redirected to the session's temporary upper layer.
func SetupOverlay(s *session.Session) error {

	// inject the host's /etc/resolv.conf into the overlay so that DNS resolution works inside Silo.
	//NB: COMMENTED OUT - DNS will now be provided by minimal file system
	//if err := injectDNS(s.Upper); err != nil {
	//	return err
	//}
//
	//// OverlayFS expects the lower filesystem to be the directory that normally
	//// represents the root filesystem.
	//   COMMENTED OUT - The lower layer is NO LONGER the host's root file system, but rather a minimal filesystem that contains only the files necessary to run a shell.
	//lower := "/"

	// The overlay's lower layer is now a minimal filesystem that contains only the
	// files necessary to run a shell. This is done to avoid polluting the overlay
	// with the host's root file system
	exe, err := os.Executable()
    if err != nil {
        return fmt.Errorf("get executable path: %w", err)
    }
    
    resDir := filepath.Join(filepath.Dir(exe), "res")
    
    lower := filepath.Join(
        resDir,
        "silo-images",
        "arm64",
        "ubuntu-24.04-LTS",
    )

	options := fmt.Sprintf(
		"lowerdir=%s,upperdir=%s,workdir=%s",
		lower,
		s.Upper,
		s.Work,
	)

	if err := unix.Mount(
		"overlay",
		s.Merged,
		"overlay",
		0,
		options,
	); err != nil {
		return fmt.Errorf("mount overlay: %w", err)
	}

	// setup requisite mounts inside the overlay so that the shell can see /proc, /dev, etc.
	if err := NewDefaultMountHandler().Apply(s.Merged); err != nil {
		return err
	}

	return nil
}

// injectDNS copies the host's /etc/resolv.conf into the overlay's upper layer so that DNS resolution works inside Silo.
func injectDNS(upperDir string) error {
	sourcePath := "/etc/resolv.conf"
	destinationPath := filepath.Join(upperDir, "etc", "resolv.conf")

	contents, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", sourcePath, err)
	}

	if err := os.MkdirAll(filepath.Dir(destinationPath), 0755); err != nil {
		return fmt.Errorf("create DNS directory: %w", err)
	}

	if err := os.WriteFile(destinationPath, contents, 0644); err != nil {
		return fmt.Errorf("write %s: %w", destinationPath, err)
	}

	return nil
}

// TeardownOverlay unmounts the overlay filesystem.
//
// MNT_DETACH allows teardown to proceed even if a process still has a reference
// to the mount. The shell should normally have exited before this is called.
func TeardownOverlay(s *session.Session) {
	if s == nil || s.Merged == "" {
		return
	}

	NewDefaultMountHandler().Cleanup(s.Merged)
	_ = unix.Unmount(s.Merged, unix.MNT_DETACH)
}
