package main

import (
	"flag"
	"fmt"
	"math"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"text/tabwriter"
	"time"
)

const (
	DefaultmaxConcurrentUpdates = 10
	DefaultNumberOfQueries      = 100
	RoundValues                 = 200
)

var (
	IPS                    = IPsMap{}
	NumberFailed     int32 = 0
	Total            int32 = 0
	NumberHostFailed int32 = 0

	maxConcurrentUpdates int
	numberOfQueries      int
	target               string
)

func init() {
	flag.IntVar(&maxConcurrentUpdates, "concurrent", DefaultmaxConcurrentUpdates, "")
	flag.IntVar(&numberOfQueries, "queries", DefaultNumberOfQueries, "")
	flag.StringVar(&target, "target", "", "")
}

func SearchHost(w *sync.WaitGroup, host string) {
	defer w.Done()

	hosts, err := net.LookupHost(host)
	if err != nil {
		atomic.AddInt32(&NumberFailed, 1)
		return
	}

	if len(hosts) == 0 {
		atomic.AddInt32(&NumberHostFailed, 1)
		return
	}
	atomic.AddInt32(&Total, 1)
	IPS.AddOrIncreaseIP(hosts[0])
}

func main() {
	flag.Parse()
	if target == "" {
		fmt.Printf("Not a valid target")
		os.Exit(1)
	}
	timmings := timmingsSlice{}
	lock := sync.RWMutex{}
	var wg sync.WaitGroup
	wg.Add(numberOfQueries)
	sem := make(chan bool, maxConcurrentUpdates)

	for i := 0; i < numberOfQueries; i++ {
		go func() {
			defer func() {
				<-sem
			}()
			sem <- true
			start := time.Now()
			SearchHost(&wg, target)
			result := round(
				float32(time.Since(start)/time.Millisecond),
				RoundValues)
			lock.Lock()
			timmings.Add(result)
			lock.Unlock()
		}()
	}
	wg.Wait()

	fmt.Println("Total: ", Total)
	fmt.Println("Number Failed: ", NumberFailed)
	fmt.Println("Number Host Failed: ", NumberHostFailed)

	IPS.Print()
	timmings.Print()
}

type IPsMap struct {
	sync.Map
	lock sync.RWMutex
}

func (ips *IPsMap) AddOrIncreaseIP(ip string) int {
	ips.lock.Lock()
	defer ips.lock.Unlock()
	counter, ok := ips.Load(ip)
	if !ok {
		ips.Store(ip, 1)
		return 1
	}
	value := counter.(int) + 1
	ips.Store(ip, value)
	return value
}

func (ips *IPsMap) Print() {
	ips.Range(func(k, v interface{}) bool {
		fmt.Printf("IP:%s, value: %v \n", k, v)
		return true
	})
	fmt.Println()
}

type timmingsSlice struct {
	data []float32
	lock sync.Mutex
}

func (t *timmingsSlice) Add(val float32) {
	t.lock.Lock()
	t.data = append(t.data, val)
	t.lock.Unlock()
}

func (t *timmingsSlice) Print() {
	results := map[float32]int{}
	for _, v := range t.data {
		results[v]++
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintln(w, "Delay Seconds\tAttempts")

	for k, v := range results {
		fmt.Fprintf(w, "%v\t%v\n", k, v)
	}
	w.Flush()
}

func round(x, unit float32) float32 {
	return float32(math.Round(float64(x)/float64(unit)) * float64(unit))
}
