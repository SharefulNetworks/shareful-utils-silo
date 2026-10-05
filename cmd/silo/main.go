//go:build linux

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/SharefulNetworks/shareful-utils-silo/internal/filesystem"
	"github.com/SharefulNetworks/shareful-utils-silo/internal/namespace"
	"github.com/SharefulNetworks/shareful-utils-silo/internal/session"
)

func main() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "silo: the application must be run as root")
		fmt.Fprintln(os.Stderr, "usage: sudo ./silo")
		os.Exit(1)
	}

	args := os.Args[1:]
	if len(args) == 0 {
		if err := runTransientShell(); err != nil {
			fatal(err)
		}

		return
	}

	switch args[0] {
	case "create":
		if len(args) != 2 {
			usageAndExit()
		}

		s, err := session.Create(args[1])
		if err != nil {
			fatal(err)
		}

		fmt.Printf("Created Silo environment %q\n", s.Name)
	case "enter":
		if len(args) != 2 {
			usageAndExit()
		}

		s, err := session.Enter(args[1])
		if err != nil {
			fatal(err)
		}
		defer s.Cleanup()

		if err := runShell(s); err != nil {
			fatal(err)
		}
	case "destroy":
		if len(args) != 2 {
			usageAndExit()
		}

		if err := session.Destroy(args[1]); err != nil {
			fatal(err)
		}

		fmt.Printf("Destroyed Silo environment %q\n", args[1])
	default:
		usageAndExit()
	}
}

func runTransientShell() error {
	s, err := session.NewTransient()
	if err != nil {
		return err
	}
	defer s.Cleanup()

	return runShell(s)
}

func runShell(s *session.Session) error {
	cleanup := func() {
		filesystem.TeardownOverlay(s)
		if s != nil && s.Transient {
			s.Cleanup()
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	defer cleanup()

	if err := namespace.EnterMountNamespace(s); err != nil {
		return err
	}

	if err := filesystem.SetupOverlay(s); err != nil {
		return err
	}
	defer filesystem.TeardownOverlay(s)

	fmt.Println("Entering Silo...")
	fmt.Println("Changes made inside this shell are isolated in the overlay.")
	fmt.Println("Type 'exit' to leave.")

	go func() {
		<-sigCh
		cleanup()
		os.Exit(1)
	}()

	if err := namespace.RunShell(s); err != nil {
		return err
	}

	return nil
}

func usageAndExit() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  silo - creates and runs an ephemeral, isolated environment, the environment (and all changes made within it) are automatically discarded on exit")
	fmt.Fprintln(os.Stderr, "  silo create NAME - create a new named, isolated environment. The environment persists until explicitly destroyed.")
	fmt.Fprintln(os.Stderr, "  silo enter NAME - enter an existing named, isolated environment")
	fmt.Fprintln(os.Stderr, "  silo destroy NAME - destroy an existing named, isolated environment")
	os.Exit(1)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "silo: %v\n", err)
	os.Exit(1)
}
