package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/daniellowtw/protocol-exercise/server/foopb"
	"google.golang.org/grpc"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"
)

var sendInterval time.Duration
var debug bool

func init() {
	flag.BoolVar(&debug, "debug", false, "print debug lines.")
	flag.DurationVar(&sendInterval, "interval", time.Second, "print debug lines.")
}

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		fmt.Println("Please provide port number.")
		os.Exit(1)
	}

	var portStr = flag.Args()[0]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Println("Port is not a number.")
		os.Exit(1)
	}
	if err := runServer(context.Background(),
		sendInterval,
		30, // 30s as specified by requirement.
		port,
		debug,
		time.Now().Unix(),
	); err != nil {
		log.Fatal(err)
	}
}

// runServer is an entrypoint so that we run this in a test.
func runServer(ctx context.Context, sendInterval time.Duration, expirationDurationSec int64, port int, debug bool, rngSeed int64) (err error) {
	rand.Seed(rngSeed)
	srv := grpc.NewServer()
	serviceImpl := &FooService{
		store: &inMemoryStore{
			sessMap: make(map[string]Session),
		},
		sendInterval:          sendInterval,
		expirationDurationSec: expirationDurationSec,
		debugMode:             debug,
		rngSrc:                rand.New(rand.NewSource(rngSeed)),
	}
	foopb.RegisterFooServer(srv, serviceImpl)
	address := fmt.Sprintf("localhost:%d", port)
	listen, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		if err = srv.Serve(listen); err != nil {
			return
		}
	}()
	select {
	case <-doneCh:
		serviceImpl.printDebug("Server is shutting down.")
	case <-ctx.Done():
		serviceImpl.printDebug("Context is finished.")
	}
	return nil
}
