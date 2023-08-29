package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

const usage = `Usage: nsqload [OPTIONS] <url>

Options:
  -c number of concurrent producers
  -r rate in messages per second per producer
  -s size of each message in bytes
`

var (
	numProducersFlag = flag.Uint("c", 1, "number of concurrent producers")
	rateFlag         = flag.Uint("r", 100, "number of messages per second to target per producer")
	messageSizeFlag  = flag.Uint("s", 1024, "size in bytes of each message")
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()

	if flag.NArg() < 1 {
		exitUsage()
	}

	nsqdURL := flag.Arg(0)
	_, err := url.ParseRequestURI(nsqdURL)
	if err != nil {
		exitUsage(fmt.Sprintf("expected -nsqd %s to be valid URL: %s", nsqdURL, err))
	}
	if *rateFlag == 0 {
		exitUsage("expected non-zero -rate")
	}
	if *messageSizeFlag == 0 {
		exitUsage("expected non-zero -size")
	}

	runProducers(nsqdURL, int(*numProducersFlag), int(*rateFlag), int(*messageSizeFlag))

}

func exitUsage(s ...any) {
	fmt.Fprintln(os.Stderr, s...)
	flag.Usage()
	os.Exit(1)
}

func runProducers(nsqd string, producers, rate, size int) {
	b64bytes := base64.StdEncoding.DecodedLen(size)
	seed := make([]byte, b64bytes)
	_, err := rand.Read(seed)
	if err != nil {
		log.Fatalf("could not create seed for random message: %s", err)
	}
	body := make([]byte, size)
	base64.StdEncoding.Encode(body, seed)
	var wg sync.WaitGroup
	for i := 0; i < producers; i++ {
		seed = append(seed[b64bytes-1:], seed[:b64bytes-1]...)
		wg.Add(1)
		go func(message []byte) {
			defer wg.Done()
			ticker := time.Tick(time.Second / time.Duration(rate))
			for {
				req, _ := http.NewRequest(http.MethodPost, nsqd, bytes.NewReader(message))
				<-ticker
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Printf("error: %v", err)
					continue
				}
				if resp.StatusCode != http.StatusOK {
					log.Printf("error: %v", resp)
				}

			}
		}(body)
	}
	wg.Wait()
}
