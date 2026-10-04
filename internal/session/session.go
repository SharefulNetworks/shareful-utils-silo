// Package session manages the filesystem used by an EmptyShell session.
package session

import (
	"fmt"
	"os"
	"path/filepath"
)

const namedEnvironmentRoot = "/var/lib/emptyshell/es_envs"

type Session struct {
	// Root is the directory containing the session's writable state.
	Root string

	// Upper is the OverlayFS upper directory. Writes made by the shell go here.
	Upper string

	// Work is required by OverlayFS and must be on the same filesystem as Upper.
	Work string

	// Merged is the directory where the overlay filesystem is mounted.
	Merged string

	// Transient indicates whether the session should be removed on cleanup.
	Transient bool

	// Name is the persistent environment name, if any.
	Name string
}

// NewTransient creates the directory layout for a new disposable EmptyShell session.
//
// This prototype deliberately uses /tmp. Later versions can move persistent
// named sessions into a dedicated directory under the user's home directory.
func NewTransient() (*Session, error) {
	root, err := os.MkdirTemp("", "emptyshell-")
	if err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}

	return newSession(root, true, "", true)
}

// New preserves the current transient behavior for older callers.
func New() (*Session, error) {
	return NewTransient()
}

// Create creates a persistent named environment under the well-known base path.
func Create(name string) (*Session, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(namedEnvironmentRoot, 0755); err != nil {
		return nil, fmt.Errorf("create environment root: %w", err)
	}

	root := filepath.Join(namedEnvironmentRoot, name)
	if err := os.Mkdir(root, 0755); err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("environment %q already exists", name)
		}

		return nil, fmt.Errorf("create environment %q: %w", name, err)
	}

	return newSession(root, false, name, true)
}

// Enter opens an existing named environment.
func Enter(name string) (*Session, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}

	root := filepath.Join(namedEnvironmentRoot, name)
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("environment %q does not exist", name)
		}

		return nil, fmt.Errorf("open environment %q: %w", name, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("environment %q is not a directory", name)
	}

	return newSession(root, false, name, false)
}

// Destroy removes a persistent named environment.
func Destroy(name string) error {
	if err := validateName(name); err != nil {
		return err
	}

	return os.RemoveAll(filepath.Join(namedEnvironmentRoot, name))
}

func newSession(root string, transient bool, name string, cleanupOnError bool) (*Session, error) {
	s := &Session{
		Root:      root,
		Upper:     filepath.Join(root, "upper"),
		Work:      filepath.Join(root, "work"),
		Merged:    filepath.Join(root, "merged"),
		Transient: transient,
		Name:      name,
	}

	for _, dir := range []string{s.Upper, s.Work, s.Merged} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			if cleanupOnError {
				_ = os.RemoveAll(root)
			}

			return nil, fmt.Errorf("create %s: %w", dir, err)
		}
	}

	return s, nil
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("environment name cannot be empty")
	}

	if filepath.Base(name) != name || name == "." || name == ".." {
		return fmt.Errorf("invalid environment name %q", name)
	}

	return nil
}

// Cleanup removes the temporary session directory.
func (s *Session) Cleanup() {
	fmt.Printf("Cleaning up transient EmptyShell session %q\n", s.Root)
	if s == nil || !s.Transient || s.Root == "" {
		return
	}

	_ = os.RemoveAll(s.Root)
}
