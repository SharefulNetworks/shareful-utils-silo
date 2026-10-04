# EmptyShell

> Instant, disposable Linux environments, like Python venv for your entire dev environment.

**EmptyShell gives you instant, disposable Linux workspaces without the weight of containers or the friction of managing images. Start clean isolated environments in seconds that are automatically cleaned up on exit** or use **named workspaces** for longer, on-going sessions, that persist until explicitly destroyed. In either case you can be confident that changes stay completely stay isolated from your host system. It’s a **super simple way to try new tools, install packages, and test projects** in a safe, disposable environment.





## Features

The current standout features of EmptyShell are as follows:


- Instant creation of transient environments that are automatically cleaned up when the shell exits
- Instant creation of named environments that can be created, entered and re-entered at any time. These will persist on disk until explicitly destroyed and are perfect for long-running development sessions.
- Compatible with common package managers and build tools, including `apt-get`, `npm`, `maven`, `pip`, `go-mod`, `cargo`, and others. Changes made to the filesystem during development session are isolated from the host system and are removed when the session ends or is explicitly destroyed.
- Support for running GUI application inside the isolated environment, including browsers, IDEs, and other graphical tools. 
- No config files, images or containers to manage. An interactive environment is created on-the-fly, that's isolated from the host filesystem.
- Super fast startup time, with no image download or container runtime overhead
- Minimal dependencies, with no need for Docker, Podman, or other container runtimes


## Requirements

- Linux
- root privileges (`sudo`)
- OverlayFS support
- Go 1.25 or newer

**Note:** EmptyShell requires root privilleges in order to support low-level filesystem operations, including mount namespaces, file system overlays,etc. The current implementation does not support unprivileged users, but this is a potential future enhancement.

## Build

```bash
go mod tidy
go build -o emptyshell ./cmd/emptyshell
```

## Run

Start a transient session:

```bash
sudo ./emptyshell
```

Create a named environment:

```bash
sudo ./emptyshell create test
```

Enter a named environment:

```bash
sudo ./emptyshell enter test
```

Inside the shell, try:

```bash
pwd
touch test-file
mkdir test-directory
echo "hello" > test-file
ls -la
```

Then exit:

```bash
exit
```

The transient session is removed when EmptyShell exits. Named environments stay
on disk until you remove them explicitly:

```bash
sudo ./emptyshell destroy test
```

## Development

A great sanity check during any development phase is to run the end-to-end lifecycle check, which creates a transient environment, installs a package, destroys it, and checks whether the root disk usage decreases as expected

The end-to-end lifecycle check requires `expect` to automate the interactive
shell session:

```bash
sudo apt-get install -y expect
```

Then run the reclaim smoke test from the repository root:

```bash
cd /path/to/shareful-utils-emptyshell
sudo ./integration/reclaim_test.sh
```

This smoke test creates an environment, installs a package, destroys it, and
checks whether the root disk usage decreases as expected. This will help ensure that any developmental changes to the codebase do not introduce bugs that break the core functionality of the tool.

## Limitation

### General 
The shell starts with a snapshot of the host filesystem, thus file system changes made to the host after the shell is started will not be visible inside the shell. This is an intentional limitation of the current implementation.

### Snap 
EmptyShell provides a virtual file system to applications ran inside the isolated environment, making it a good fit for many package installs that primarily write
into the filesystem, including tools like `apt-get`, `npm`, `maven`, and other
build-time dependency managers. However, some package managers, like `snap`, require deep integration with the host system and are not compatible with the current implementation of EmptyShell.

### GUI Applications
EmptyShell does supports running GUI applications inside the isolated environment, including browsers, IDEs, and other graphical tools. However, some GUI applications may require additional configuration or dependencies to run properly inside the isolated environment. Additionally it's good practice to start GUI applications with `&` e.g 

```bash
firefox &
```
This will start the application in the background and allow you to continue using the isolated shell without blocking the terminal.

**EmptyShell, the Python venv for your entire dev environment.**
