package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	var in, out chan interface{}
	wg := &sync.WaitGroup{}

	for _, j := range jobs {
		in = out
		out = make(chan interface{}, 100)

		wg.Add(1)
		go func(j job, in, out chan interface{}) {
			defer wg.Done()
			defer close(out)

			j(in, out)
		}(j, in, out)
	}

	wg.Wait()
}

func processPipeItem(in, out chan interface{}, f func(string) string) {
	wg := &sync.WaitGroup{}

	for inItem := range in {
		wg.Add(1)

		go func(inItem interface{}) {
			defer wg.Done()

			out <- f(fmt.Sprintf("%v", inItem))
		}(inItem)
	}

	wg.Wait()
}

func SingleHash(in chan interface{}, out chan interface{}) {
	md5Mux := &sync.Mutex{}

	processPipeItem(in, out, func(data string) string {
		hash1 := make(chan string)
		go func() {
			defer close(hash1)

			hash1 <- DataSignerCrc32(data)
		}()

		md5Mux.Lock()
		md5 := DataSignerMd5(data)
		md5Mux.Unlock()

		hash2 := DataSignerCrc32(md5)
		return fmt.Sprintf("%s~%s", <-hash1, hash2)
	})
}

func MultiHash(in chan interface{}, out chan interface{}) {

	processPipeItem(in, out, func(data string) string {
		results := make([]string, 6)

		resMutex := &sync.Mutex{}
		resWg := &sync.WaitGroup{}

		for th := 0; th <= 5; th++ {
			resWg.Add(1)

			go func(th int) {
				defer resWg.Done()

				thRes := DataSignerCrc32(fmt.Sprintf("%d%s", th, data))
				resMutex.Lock()
				results[th] = thRes
				resMutex.Unlock()
			}(th)
		}

		resWg.Wait()
		return strings.Join(results, "")
	})
}

func CombineResults(in chan interface{}, out chan interface{}) {
	var results []string

	for data := range in {
		results = append(results, fmt.Sprintf("%v", data))
	}

	sort.Strings(results)
	out <- strings.Join(results, "_")
}
