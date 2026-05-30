// Command airan is the env for AI coding agents: a shebang dispatcher
// that resolves an agent file to a concrete agent CLI and execs it.
//
// Usage:
//
//	airan FILE
//
// Typically invoked via a shebang rather than directly:
//
//	#!/usr/bin/env airan
//	---
//	backend: claude
//	---
//	<prompt body>
//
// See package github.com/kfet/airan for the resolution and adapter
// logic.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/kfet/airan"
)

func main() {
	if err := airan.Run(os.Args[1:], os.Getenv, os.Environ(), os.Stdout, execProcess); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// execProcess looks up cmd on $PATH and replaces the current process
// with it via execve. On success it does not return.
func execProcess(cmd string, args []string, environ []string) error {
	path, err := exec.LookPath(cmd)
	if err != nil {
		return err
	}
	argv := append([]string{cmd}, args...)
	return syscall.Exec(path, argv, environ)
}
