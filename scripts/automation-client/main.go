// automation-client bridges stdin/stdout to MyGo's Unix socket on Windows,
// where some Python distributions do not provide socket.AF_UNIX.
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: automation-client socket")
		os.Exit(1)
	}
	var conn net.Conn
	var err error
	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		conn, err = net.DialTimeout("unix", os.Args[1], time.Second)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()
	requests, responses := bufio.NewScanner(os.Stdin), bufio.NewReader(conn)
	requests.Buffer(make([]byte, 4096), 8<<20)
	for requests.Scan() {
		conn.SetDeadline(time.Now().Add(30 * time.Second))
		if _, err := fmt.Fprintln(conn, requests.Text()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		line, err := responses.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print(line)
	}
}
