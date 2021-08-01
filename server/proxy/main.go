package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

var (
	durationToRun time.Duration
	// portSrc is the port to listen for incoming conn.
	portSrc int
	// portSrc is the port to dial to for each incoming conn.
	portDst int
)

func init() {
	flag.DurationVar(&durationToRun, "duration", 0, "how long to run the proxy for")
	flag.IntVar(&portSrc, "src", 10241, "port to listen on")
	flag.IntVar(&portDst, "dst", 10240, "port to dial to")
}

// This can be run with go run to run this proxy on the terminal.
func main() {
	flag.Parse()
	if durationToRun != 0 {
		fmt.Println(durationToRun)
		go func() {
			<-time.Tick(durationToRun)
			os.Exit(0)
		}()
	}
	fmt.Println(durationToRun)
	runProxy(fmt.Sprintf("localhost:%d", portSrc),
		fmt.Sprintf("localhost:%d", portDst),
	)
}

// runProxy listens on srcAddr for tcp connections and forwards them to dstAddr.
func runProxy(srcAddr, dstAddr string) {
	listen, err := net.Listen("tcp", srcAddr)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			return
		}
		targetConn, err := net.Dial("tcp", dstAddr)
		if err != nil {
			return
		}
		go proxy(conn, targetConn)
	}
}

func proxy(incoming, outgoing io.ReadWriter) {
	go io.Copy(incoming, outgoing)
	go io.Copy(outgoing, incoming)
}
