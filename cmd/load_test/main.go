package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type stage struct {
	users    int
	duration time.Duration
}

var (
	targetAddr      = flag.String("target-addr", "localhost:8000", "")
	pprofURL        = flag.String("pprof-url", "http://localhost:6060/debug/pprof", "")
	reqsPerSec      = flag.Int("reqs-per-sec", -1, "")
	concurrentConns = flag.Int("concurrent-conns", -1, "")

	start     = time.Now()
	heapAlloc string
	latency   time.Duration

	mu    = &sync.RWMutex{}
	conns int
)

func main() {
	flag.Parse()
	errChan := make(chan error, 1)

	go func() {
		if err := <-errChan; err != nil {
			slog.Error(err.Error())
		}
		os.Exit(1)
	}()

	if *concurrentConns != -1 && *reqsPerSec == -1 {
		for i := range *concurrentConns {
			pingInterval := time.Second * time.Duration((i%2)+1)
			go func() {
				errChan <- errors.Wrap(handleNewConn(pingInterval), "cannot handle TCP connection to target")
			}()
		}
	}

	go func() { errChan <- errors.Wrap(getHeapAlloc(), "cannot get heap alloc") }()
	go func() { errChan <- errors.Wrap(getLatency(), "cannot get latency") }()

	ticker := time.NewTicker(time.Second / 2)
	fmt.Println("Time,Heap Alloc,Latency")

	for {
		<-ticker.C
		if conns == *concurrentConns {
			fmt.Printf("%s,%s,%s\n", time.Now().Sub(start), heapAlloc, latency)
		}
	}
}

func handleNewConn(pingInterval time.Duration) error {
	conn, err := net.Dial("tcp", *targetAddr)
	if err != nil {
		return errors.Wrapf(err, "cannot dial target with %v concurrent connections", conns)
	}

	mu.Lock()
	conns++
	mu.Unlock()

	defer func() {
		conn.Close()

		mu.Lock()
		conns--
		mu.Unlock()
	}()

	for {
		if _, err = conn.Write([]byte(`PING`)); err != nil {
			return errors.Wrapf(err, "cannot write ping message with %v concurrent connections", conns)
		}

		time.Sleep(pingInterval)
	}
}

func getHeapAlloc() error {
	for {
		res, err := http.Get(*pprofURL + "/heap?debug=1")
		if err != nil {
			return errors.Wrap(err, "request failed")
		}

		b, err := io.ReadAll(res.Body)
		if err != nil {
			return errors.Wrap(err, "cannot read response")
		}

		lines := strings.Split(string(b), "\n")
		h := lines[len(lines)-22]
		heapAlloc = strings.TrimPrefix(h, "# HeapAlloc = ")

		time.Sleep(time.Second / 2)
	}
}

func getLatency() (err error) {

	var conn net.Conn
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	conn, err = net.Dial("tcp", *targetAddr)
	if err != nil {
		return errors.Wrapf(err, "cannot dial target with %v concurrent connections", conns)
	}

	for {
		t := time.Now()
		_, err = conn.Write([]byte(`PING`))
		l := time.Now().Sub(t)

		if err != nil {
			return errors.Wrapf(err, "cannot write ping message with %v concurrent connections", conns)
		}

		latency = l
		time.Sleep(time.Second / 10)
	}
}
