package main

import (
	"context"
	"time"
	"bytes"
	"os/exec"
	"io"
	"fmt"
)

func execCmd(bin string, stdin string, args []string, timeout int) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout) * time.Second)
	defer cancel()

	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stderr = &errBuffer
	cmd.Stdout = &outBuffer

	if stdin != "" {
		stdinPipe, errStdin := cmd.StdinPipe()
		if errStdin != nil {
			return "", "", errStdin
		}
		defer stdinPipe.Close()
		io.WriteString(stdinPipe, stdin + "\n")
	}

	err := cmd.Run()
	if err != nil {
		return outBuffer.String(), errBuffer.String(), err
	}

	return outBuffer.String(), errBuffer.String(), nil
}

func dumpCmdResult(stdout, stderr string) {
	if stdout != "" {
		warnLog("dump-cmd-stdout start")
		fmt.Println("--------------------------------")
		fmt.Printf(stdout)
		fmt.Println("--------------------------------")
		warnLog("dump-cmd-stdout end")
	}

	if stderr != "" {
		errorLog("dump-cmd-stderr start")
		fmt.Println("--------------------------------")
		fmt.Println(stderr)
		fmt.Println("--------------------------------")
		errorLog("dump-cmd-stderr end")
	}
}
