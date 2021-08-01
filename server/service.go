package main

import (
	"fmt"
	"github.com/daniellowtw/protocol-exercise/server/foopb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/rand"
	"time"
)

type FooService struct {
	*foopb.UnimplementedFooServer

	// store to keep track of client and its state. In this implementation there's nothing cleaning up the
	// orphan clients. In production, the constructor for this struct should also spin up a goroutine to
	// traverse the store's keys and prune client ID states that have expired and can be reused.
	store store

	// How long to send between polls.
	sendInterval time.Duration

	// Number of seconds before state is stale.
	expirationDurationSec int64

	debugMode bool

	// RngSrc is the random number generator source.
	rngSrc *rand.Rand
}

func (f *FooService) printDebug(v interface{}) {
	if f.debugMode {
		fmt.Printf("server: %+v\n", v)
	}
}

// Stateless assumes that the total number of expected messages is smaller than 32. Because after that the message
// sent back will be 0 and is indistinguishable from if the total message had be some other number.
func (f *FooService) Stateless(req *foopb.StatelessRequest, server foopb.Foo_StatelessServer) error {
	f.printDebug(req)
	numberToSend := req.LastMsg * 2

	t := time.NewTicker(f.sendInterval)
	defer t.Stop()

	if numberToSend == 0 {
		b := make([]byte, 1)
		f.rngSrc.Read(b)
		numberToSend = uint32(b[0])
	}

	for {
		err := server.Send(&foopb.StatelessResponse{Msg: numberToSend})
		if err != nil {
			if status.Code(err) == codes.Unavailable {
				return nil
			}
			if status.Code(err) == codes.Canceled {
				return nil
			}
			f.printDebug(fmt.Sprintf("cannot send: %s", err))
		}
		f.printDebug(fmt.Sprintf("sent : %d", numberToSend))
		numberToSend *= 2
		<-t.C
	}
}

func (f *FooService) Stateful(req *foopb.StatefulRequest, server foopb.Foo_StatefulServer) error {
	f.printDebug(req)

	// Assumes well behaved client.
	var sess Session
	if req.IsReconnect { // Retrieve existing state.
		innerSess, err := f.store.Read(req.ClientId)
		if err != nil {
			return status.Errorf(codes.Internal, "cannot establish session store:%s", err)
		}
		if time.Now().Unix()-innerSess.lastActivityEpoch > f.expirationDurationSec {
			return status.Errorf(codes.FailedPrecondition, "retry without reconnect.")
		}
		sess = innerSess
	} else { // Create new state.
		seed := int64(f.rngSrc.Uint64())
		sess = Session{
			seed:              seed,
			progress:          0,
			totalMsg:          req.TotalMsg,
			lastActivityEpoch: time.Now().Unix(),
		}
		if err := f.store.Store(req.ClientId, sess); err != nil {
			return status.Errorf(codes.Internal, "cannot store state")
		}
	}

	rng := rand.New(rand.NewSource(sess.seed))

	// Catch up to the progress. Would be no-op for new requests.
	var checkSum uint32
	var msgSent uint32 = 0
	for ; msgSent < sess.progress; msgSent++ {
		checkSum += rng.Uint32()
	}

	t := time.NewTicker(f.sendInterval)
	defer t.Stop()

	for ; msgSent < sess.totalMsg; msgSent++ {
		msgToSend := rng.Uint32()
		// Checksum is a simple sum here that doesn't check out of order messages for simplicity.
		// In production, a sum function like checkSum = 31 * msgToSend + checkSum can be used.
		checkSum += msgToSend

		sess.progress++
		sess.lastActivityEpoch = time.Now().Unix()
		if err := f.store.Store(req.ClientId, sess); err != nil {
			return status.Errorf(codes.Internal, "cannot store state")
		}

		var checksumToSend uint32
		if msgSent == sess.totalMsg-1 {
			checksumToSend = checkSum
		}

		resp := &foopb.StatefulResponse{
			Msg:      msgToSend,
			Checksum: checksumToSend,
		}
		f.printDebug(resp)
		if err := server.Send(resp); err != nil {
			f.printDebug(err)
			if status.Code(err) == codes.Unavailable {
				return nil
			}
		}
		<-t.C
	}

	if err := f.store.Delete(req.ClientId); err != nil {
		f.printDebug(err)
	}
	return nil
}
