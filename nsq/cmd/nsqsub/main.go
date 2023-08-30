package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/aybabtme/uniplot/histogram"
	"github.com/nsqio/go-nsq"
)

const usage = `Usage: nsqsub [OPTIONS] <lookupd url> <topic> <channel>

Options:
  -c number of concurrent subscribers
  -d delay between accepting a message and calling FIN
  -t time to run. Default 30s
  -localaddr optionally selects local address to use
  -o file name to write report to
`

var (
	numConsumersFlag = flag.Uint("c", 1, "number of concurrent consumers")
	finDelayFlag     = flag.Duration("d", 0, "delay between accepting a message and calling FIN")
	timeFlag         = flag.Duration("t", time.Second*30, "time to run")
	localAddrFlag    = flag.String("localaddr", "", "optionally selects local address to use")
	outputFlag       = flag.String("o", "", "report output file")
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()

	if flag.NArg() < 2 {
		exitUsage()
	}

	lookupURL := flag.Arg(0)
	_, err := url.ParseRequestURI(lookupURL)
	if err != nil {
		exitUsage(fmt.Sprintf("expected -nsqd %s to be valid URL: %s", lookupURL, err))
	}

	var localAddr net.Addr
	if *localAddrFlag != "" {
		ip := net.ParseIP(*localAddrFlag)
		if ip == nil {
			exitUsage(fmt.Sprintf("could not parse localaddr as ip %s", *localAddrFlag))
		}
		localAddr = &net.TCPAddr{
			IP:   ip,
			Port: 0,
		}
	}

	topic := flag.Arg(1)
	if topic == "" {
		exitUsage("expected non-empty topic name")
	}
	channel := flag.Arg(2)
	if channel == "" {
		exitUsage("expected non-empty channel name")
	}

	outfile := os.Stdout
	if *outputFlag != "" {
		f, err := os.OpenFile(*outputFlag, os.O_WRONLY|os.O_CREATE, 0664)
		if err != nil {
			exitUsage(fmt.Sprintf("could not open %s: %s", *outputFlag, err))
		}
		defer f.Close()
		outfile = f
	}

	numConsumers := int(*numConsumersFlag)
	var cReadyWG, cRunningWG sync.WaitGroup
	resultC := make(chan int64, numConsumers)
	begin := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config := nsq.NewConfig()

	sharedLookupdClient := &http.Client{
		Timeout: time.Minute,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				LocalAddr: localAddr,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			ExpectContinueTimeout: time.Second,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		},
	}

	config.DialTimeout = 10 * time.Second
	if localAddr != nil {
		config.LocalAddr = localAddr
	}
	for i := 0; i < numConsumers; i++ {
		cRunningWG.Add(1)
		cReadyWG.Add(1)
		go func() {
			var ct atomic.Int64
			ct.Store(0)
			defer cRunningWG.Done()
			defer func() {
				resultC <- ct.Load()
			}()
			consumer, err := nsq.NewConsumer(topic, channel, config)
			if err != nil {
				log.Fatalf("error creating new nsq consumer: %s", err)
			}

			consumer.SetLookupdHttpClient(sharedLookupdClient)
			consumer.AddHandler(&handler{
				Delay: *finDelayFlag,
				IncFn: func() { ct.Add(1) },
			})
			cReadyWG.Done()
			<-begin
			err = consumer.ConnectToNSQLookupd(lookupURL)
			if err != nil {
				log.Fatalf("error connecting consumer to nsq lookupd: %s", err)
			}
			<-ctx.Done()
			consumer.Stop()

		}()
	}

	log.Printf("connected to %s with %d concurrent consumers", lookupURL, *numConsumersFlag)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	cReadyWG.Wait()
	deadline := time.After(*timeFlag)
	close(begin)
	go func() {
		select {
		case <-shutdown:
			log.Println("received shutdown signal. Stoping consumers.")
		case <-deadline:
		}
		cancel()
	}()

	cRunningWG.Wait()
	close(resultC)
	var results []float64
	for i := range resultC {
		results = append(results, float64(i))
	}

	dist := histogram.Hist(10, results)
	histogram.Fprintf(outfile, dist, histogram.Linear(80), func(v float64) string {
		// should be whole decimal numbers
		return fmt.Sprintf("%d", int(v))
	})
}

func exitUsage(s ...any) {
	fmt.Fprintln(os.Stderr, s...)
	flag.Usage()
	os.Exit(1)
}

type handler struct {
	Delay time.Duration
	IncFn func()
}

func (h *handler) HandleMessage(msg *nsq.Message) error {
	<-time.After(h.Delay)
	msg.Finish()
	h.IncFn()
	return nil
}
