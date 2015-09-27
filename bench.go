package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	workload   = 1
	urls       = make(chan string, 10000)
	adhockUrls = make(chan string, 1000)
	points     = make(chan int, 1000)
	score      = 0
)

func parse(response *http.Response) {
	if response.StatusCode >= 400 {
		points <- -1
	} else if response.StatusCode >= 300 {
		points <- 1
	} else {
		points <- 10
	}
	// if url is parsed
	// adhockUrls <- new_url
}

func request(url string) {
	resp, err := http.Get(url)
	if err != nil {
		points <- -1
		return
	}
	go parse(resp)
}

func worker(id int) {
	// adhock_url is higher priority
	select {
	case url := <-adhockUrls:
		request(url)
	case url := <-urls:
		request(url)
	}
}

func scorer() {
	score += <-points
}

func bench() int {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		urls <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln("reading standard input:", err)
		return 1
	}
	timer := time.NewTimer(time.Second * 60)
	go scorer()
	for w := 0; w < workload; w++ {
		go worker(w)
	}
	<-timer.C
	fmt.Println("Your score is: ", score)
	return 0
}

func main() {
	os.Exit(bench())
}
