package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func main() {
	inputData := []int{0, 1, 2, 3, 4, 5, 6}
	var in = make(chan interface{}, len(inputData))
	var out = make(chan interface{})
	for _, i := range inputData {
		in <- i
	}
	close(in)
	go func() {
		defer close(out)
		ExecutePipeline(
			func(in, out chan interface{}) {
				for i := range in {
					out <- i
				}
			},
			SingleHash,
			MultiHash,
			CombineResults,
		)
	}()
	for result := range out {
		fmt.Println("Result:", result)
	}
}

func ExecutePipeline(freeFlowJobs ...job) {
	var in = make(chan interface{})
	var wg = &sync.WaitGroup{}

	for _, j := range freeFlowJobs {
		var out = make(chan interface{})
		wg.Add(1)
		go func(j job, in, out chan interface{}) {
			defer wg.Done()
			j(in, out)
			close(out)
		}(j, in, out)
		in = out
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	var wg = &sync.WaitGroup{}
	var mutex = &sync.Mutex{}

	for inputData := range in {
		wg.Add(1)
		go func(inputData interface{}) {
			defer wg.Done()
			data := strconv.Itoa(inputData.(int))
			mutex.Lock()
			hashMd5 := DataSignerMd5(data)
			mutex.Unlock()

			var crc32 = make(chan string)
			go func() {
				crc32 <- DataSignerCrc32(data)
			}()

			crc32md5 := DataSignerCrc32(hashMd5)
			crc32data := <-crc32

			out <- crc32data + "~" + crc32md5
		}(inputData)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	var wg = &sync.WaitGroup{}

	for inputData := range in {
		wg.Add(1)
		go func(inputData interface{}) {
			defer wg.Done()
			data := inputData.(string)
			var result [6]string
			var wgData = &sync.WaitGroup{}

			for i := 0; i < 6; i++ {
				wgData.Add(1)
				go func(i int) {
					defer wgData.Done()
					result[i] = DataSignerCrc32(strconv.Itoa(i) + data)
				}(i)
			}
			wgData.Wait()
			out <- strings.Join(result[:], "")
		}(inputData)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	var results []string
	for data := range in {
		results = append(results, data.(string))
	}
	sort.Strings(results)
	out <- strings.Join(results, "_")
}
