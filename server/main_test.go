package main

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStatelessClientNoReconnect(t *testing.T) {
	ctx, cancelF := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelF()
	go func() {
		if err := runServer(ctx, time.Millisecond, 1, 10240, true, 2); err != nil {
			t.Errorf("server: %s\n", err)
		}
	}()
	// Assumes node js runtime in env
	cmd := exec.CommandContext(ctx, "node", "index.js", "--port", "10240", "--type", "stateless", "10")
	cmd.Dir = "../client"
	var clientOut bytes.Buffer
	cmd.Stderr = &clientOut
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "48081" {
		t.Errorf("The sum for stateless mode with the fixed rng seed is not correct. Expected (48081) vs Actual(%s)", string(out))
		t.Fail()
	}
}

func TestStatelessClientReconnect(t *testing.T) {
	ctx, cancelF := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelF()
	go func() {
		if err := runServer(ctx, 100*time.Millisecond, 1, 10242, true, 2); err != nil {
			t.Errorf("server: %s\n", err)
		}
	}()
	go runProxy(ctx, t, time.Millisecond*300, 10243, 10242)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Assumes node js runtime in env
		cmd := exec.CommandContext(ctx, "node", "index.js", "--port", "10243", "--type", "stateless", "10")
		cmd.Dir = "../client"
		var clientOut bytes.Buffer
		cmd.Stderr = &clientOut
		out, err := cmd.Output()
		if err != nil {
			t.Error(err)
		}
		if strings.TrimSpace(string(out)) != "48081" {
			t.Errorf("The sum for stateless mode with the fixed rng seed is not correct. Expected (48081) vs Actual(%s)", string(out))
			t.Fail()
		}
	}()
	// Simulate network issue by waiting for the TCP proxy to stop
	time.Sleep(2 * time.Second)
	go runProxy(ctx, t, time.Second* 5, 10243, 10242)

	wg.Wait()
}

func TestStatefulClientNoReconnect(t *testing.T) {
	ctx, cancelF := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelF()
	go func() {
		if err := runServer(ctx, time.Millisecond, 1, 10244, true, 2); err != nil {
			t.Errorf("server: %s\n", err)
		}
	}()
	// Assumes node js runtime in env
	cmd := exec.Command("node", "index.js", "--port", "10244", "--type", "stateful", "--id", "123", "--total-overwrite", "10")
	cmd.Dir = "../client"
	var clientOut bytes.Buffer
	cmd.Stderr = &clientOut
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "Checksum matches" {
		t.Fatalf("failed. Expecting checksum to match: %s", out)
	}
}

func TestStatefulClientReconnect(t *testing.T) {
	ctx, cancelF := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelF()
	go runServer(ctx, 500*time.Millisecond, 30, 10248, true, 2)
	go runProxy(ctx, t, time.Millisecond*500, 10249, 10248)

	// Assumes node js runtime in env
	runClientCmd := exec.Command("node", "index.js", "--port", "10249", "--type", "stateful", "--id", "124", "--total-overwrite", "10")
	runClientCmd.Dir = "../client"
	var clientOut bytes.Buffer
	runClientCmd.Stderr = &clientOut

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		out, err := runClientCmd.Output()
		if err != nil {
			t.Error(err)
			t.Fail()
			return
		}
		if strings.TrimSpace(string(out)) != "Checksum matches" {
			t.Errorf("failed. Expecting checksum to match but output was: %s %s", string(out), clientOut.String())
			t.Fail()
			return
		}
	}()

	// Simulate network issue by waiting for the TCP proxy to stop
	time.Sleep(time.Second)

	go runProxy(ctx, t, 10*time.Second, 10249, 10248)
	wg.Wait()
}

func TestStatefulClientReconnectTooSlowly(t *testing.T) {
	ctx, cancelF := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelF()
	// The client will reconnect after the state has expired.
	go runServer(ctx, 500*time.Millisecond, 1, 10250, true, 2)

	go runProxy(ctx, t, time.Second, 10251, 10250)

	// Assumes node js runtime in env
	runClientCmd := exec.Command("node", "index.js", "--port", "10251", "--type", "stateful", "--id", "123", "--total-overwrite", "10")
	runClientCmd.Dir = "../client"
	var clientOut bytes.Buffer
	runClientCmd.Stderr = &clientOut

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		out, err := runClientCmd.Output()
		if err != nil {
			t.Error(err)
			t.Fail()
			return
		}
		if !strings.HasPrefix(strings.TrimSpace(string(out)), "Expired") {
			t.Errorf("failed. Expecting checksum to match but output was: %s %s", string(out), clientOut.String())
			t.Fail()
			return
		}
	}()

	// Simulate network issue by waiting for the TCP proxy to stop
	time.Sleep(2 * time.Second)
	go runProxy(ctx, t, 5*time.Second, 10251, 10250)
	wg.Wait()
}

// Proxies localhost:<src> -> localhost:<dst>
func runProxy(ctx context.Context, t *testing.T, duration time.Duration, srcPort int, dstPort int) {
	cmd := exec.CommandContext(ctx, "go", "run", "main.go", "-duration", duration.String(), "-src", strconv.Itoa(srcPort), "-dst", strconv.Itoa(dstPort))
	cmd.Dir = "proxy"
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(err)
	}
	if len(out) != 0 {
		t.Log(string(out))
	}
}
