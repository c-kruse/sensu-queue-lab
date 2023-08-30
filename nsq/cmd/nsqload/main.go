package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nsqio/go-nsq"
)

const usage = `Usage: nsqload [OPTIONS] <nsqd tcp address> <topic>

Options:
  -c number of concurrent producers
  -r rate in messages per second per producer
  -s size of each message in bytes
  -t time to run. Default 30s
`

var (
	numProducersFlag = flag.Uint("c", 1, "number of concurrent producers")
	rateFlag         = flag.Uint("r", 100, "number of messages per second to target per producer")
	messageSizeFlag  = flag.Uint("s", 1024, "size in bytes of each message")
	timeFlag         = flag.Duration("t", time.Second*30, "time to run")
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()

	if flag.NArg() < 2 {
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
	topic := flag.Arg(1)
	if !nsq.IsValidTopicName(topic) {
		exitUsage(fmt.Sprintf("invalid topic name %s", topic))
	}

	var cReadyWG, cRunningWG sync.WaitGroup
	begin := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config := nsq.NewConfig()

	type result struct {
		Count    int
		Duration time.Duration
	}

	results := make(chan result, int(*numProducersFlag))
	size := int(*messageSizeFlag)
	b64bytes := base64.StdEncoding.DecodedLen(size)
	msgBase := seed(b64bytes)
	body := make([]byte, size)
	for i := 0; i < int(*numProducersFlag); i++ {
		msgBase = rot(msgBase)
		base64.StdEncoding.Encode(body, msgBase)

		cRunningWG.Add(1)
		cReadyWG.Add(1)
		go func(msg []byte) {
			defer cRunningWG.Done()
			producer, err := nsq.NewProducer(nsqdURL, config)
			if err != nil {
				log.Fatal(err)
			}
			cReadyWG.Done()
			<-begin

			var count int
			var elapsed time.Duration
			defer func() {
				results <- result{count, elapsed}
			}()
			ticker := time.Tick(time.Second / time.Duration(*rateFlag))
			for {
				pubResult := make(chan *nsq.ProducerTransaction)
				<-ticker
				start := time.Now()
				producer.PublishAsync(topic, msg, pubResult)
				select {
				case <-ctx.Done():
					return
				case result := <-pubResult:
					if result.Error != nil {
						log.Printf("error publishing message: %v", err)
					}
					count++
					elapsed = elapsed + time.Since(start)
				}
			}
		}(body)
	}
	log.Printf("connected to %s with %d concurrent consumers", nsqdURL, *numProducersFlag)

	cReadyWG.Wait()
	close(begin)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	deadline := time.After(*timeFlag)
	go func() {
		select {
		case <-shutdown:
			log.Println("received shutdown signal. Stoping consumers.")
		case <-deadline:
		}
		cancel()
	}()

	go func() {
		for r := range results {
			fmt.Println(r)
		}
	}()

	cRunningWG.Wait()

}

func exitUsage(s ...any) {
	fmt.Fprintln(os.Stderr, s...)
	flag.Usage()
	os.Exit(1)
}

func seed(n int) []byte {
	seed := make([]byte, n)
	_, err := rand.Read(seed)
	if err != nil {
		log.Fatalf("could not create seed for random message: %s", err)
	}
	return seed
}

func rot(in []byte) []byte {
	return append(in[len(in)-1:], in[:len(in)-1]...)
}
