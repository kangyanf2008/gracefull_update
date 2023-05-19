package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"time"
)

var (
	graceful = flag.Bool("graceful", false, "-graceful")
)

type Accepted struct {
	conn net.Conn
	err  error
}

func listenAndServer(ln net.Listener, sig chan os.Signal) {
	accepted := make(chan Accepted, 2)
	go func() {
		for {
			conn, err := ln.Accept()
			accepted <- Accepted{
				conn: conn,
				err:  err,
			}
		}
	}()

	for {
		select {
		case act := <-accepted:
			if act.err == nil {
				fmt.Println("handle connection")
				go handleConnection(act.conn)
			}
		case s := <-sig:
			fmt.Printf("gonna fork and run, signal: %#v", s)
			forkAndRun(ln)
			break
		}
	}

}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	counter := int64(0)
	fdFile, err := conn.(*net.TCPConn).File()
	if err != nil {
		fmt.Printf("handleConnection get fdFile errr, err: %#v \n", err)
	}

	fd := fdFile.Fd()
	for true {
		counter++
		conn.Write([]byte("hello: " + strconv.FormatInt(counter, 10) + ", fd: " + strconv.FormatInt(int64(fd), 10) + "\n"))
		time.Sleep(time.Millisecond * 5000)
	}
}

func forkAndRun(ln net.Listener) {
	l := ln.(*net.TCPListener)
	fdFile, err := l.File()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("forAndRun fd: %d, osArgs: %s\n", fdFile.Fd(), os.Args[0])
	cmd := exec.Command(os.Args[0], "-graceful")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.ExtraFiles = []*os.File{fdFile}
}

func graceFullListener() net.Listener {
	ln, err := net.FileListener(os.NewFile(3, "graceful server"))
	if err != nil {
		fmt.Println(err)
	}
	return ln
}

func firstBootListener() net.Listener {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println(err)
	}
	return ln
}

func main() {
	flag.Parse()
	fmt.Printf("given args: %t, pid: %d \n", *graceful, os.Geteuid())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig)

	var ln net.Listener
	if *graceful {
		ln = graceFullListener()
	} else {
		ln = firstBootListener()
	}
	listenAndServer(ln, sig)
}
