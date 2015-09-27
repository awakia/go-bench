package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/net/html"
)

var (
	workload       = 1
	requests       = make(chan *Request, 10000)
	adhockRequests = make(chan *Request, 1000)
	points         = make(chan int, 1000)
	score          = 0
)

// RequestMethod defines http request method
type RequestMethod int

// Request methods
const (
	GET RequestMethod = iota
	POST
)

func (rm RequestMethod) String() string {
	switch rm {
	case GET:
		return "GET"
	case POST:
		return "POST"
	default:
		return "Unknown"
	}
}

// Request defines http request method and url
type Request struct {
	Method RequestMethod
	URL    string
	Values url.Values
}

// NewPostRequest creates new POST Request instance
func NewPostRequest(url string, values url.Values) *Request {
	return &Request{POST, url, values}
}

// NewGetRequest creates new GET Request instance
func NewGetRequest(url string) *Request {
	return &Request{GET, url, nil}
}

func parse(reqURL string, response *http.Response) {
	if response.StatusCode >= 400 {
		points <- -1
	} else if response.StatusCode >= 300 {
		points <- 1
	} else {
		points <- 10
	}
	parseHTML(reqURL, response.Body)
	// if new url is parsed
	// adhockRequests <- NewGetRequest(newURL)
	response.Body.Close()
}

func parseHTML(reqURL string, r io.Reader) {
	doc, err := html.Parse(r)
	if err != nil {
		fmt.Println(err)
	}
	curURL, _ := url.Parse(reqURL)

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "link":
				for _, a := range n.Attr {
					if a.Key == "href" {
						foundURL, _ := curURL.Parse(a.Val)
						requests <- NewGetRequest(foundURL.String())
					}
				}
			case "img", "script":
				for _, a := range n.Attr {
					if a.Key == "src" {
						foundURL, _ := curURL.Parse(a.Val)
						requests <- NewGetRequest(foundURL.String())
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
}

func check(request *Request) {
	var resp *http.Response
	var err error
	switch request.Method {
	case GET:
		resp, err = http.Get(request.URL)
	case POST:
		resp, err = http.PostForm(request.URL, request.Values)
	}
	if err != nil {
		points <- -1
		return
	}
	go parse(request.URL, resp)
}

func worker(id int) {
	// adhock_url is higher priority
	select {
	case request := <-adhockRequests:
		check(request)
	case request := <-requests:
		check(request)
	}
}

func scorer() {
	score += <-points
}

func bench() int {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		requests <- NewGetRequest(scanner.Text())
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
