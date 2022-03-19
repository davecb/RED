package main

// redstat is a 'stat' command for RED, requests, errors and duration

import (
	r "../../pkg/red"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	u "net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func usage() {
	fmt.Printf("Usage: %s [-v] url [delay [count]]", os.Args[0])
	os.Exit(1)
}

// main parses command-line parameters
func main() {
	var verbose, json bool
	var delay, count int
	var err error

	flag.BoolVar(&verbose, "verbose", false, "turn on verbose messages")
	flag.BoolVar(&json, "json", false, "report in json format")
	flag.Parse()

	url := flag.Arg(1)
	if url == "" {
		log.Printf("You must provide a url to send a RED request to\n")
		usage()
	}
	_, err = u.ParseRequestURI(url)
	if err != nil {
		log.Printf("url value must be a legal URL, parser reported %s\n", err)
		usage()
	}

	if d := flag.Arg(2); d != "" {
		delay, err = strconv.Atoi(d)
		if err != nil {
			log.Printf("delay value must be an int\n")
			usage()
		}
	} else {
		delay = -1
	}

	if c := flag.Arg(3); c != "" {
		count, err = strconv.Atoi(c)
		if err != nil {
			log.Printf("count value must be an int\n")
			usage()
		}
	} else {
		count = -1
	}

	_ = redstat(url, delay, count, verbose, json, false)
}

// redstat queries an interface a specified number of times
// if delay is absent, a single value is returned
// if count is absent, a continuous series of values are returned
// assumes duration is in wall-clock time
func redstat(url string, delay, count int, verbose, json, crash bool) *r.Red {
	var first, second, difference *r.Red
	var err error

	// get the first query
	// now see if we loop
	switch {
	case count == -1, count == 0:
		// just report and return
		first, err = getRed(url, verbose)
		if err != nil {
			if crash {
				panic(err)
			}
			log.Fatalf("redstat: fatal error, halting. Message was %q\n", err)
		}
		report(first, json) // duration will be (now - program start time)
		return first        // Used in testing
	case delay != -1:
		// wait, subtract and report the differences
		first, err = getRed(url, verbose)
		if err != nil {
			if crash {
				panic(err)
			}
			log.Fatalf("redstat: fatal error, halting. Message was %q\n", err)
		}
		if verbose {
			log.Printf("sample 0 was %s\n", first.String())
		}
		tick := time.Duration(delay) * time.Second
		for i := 1; i < (count + 1); i++ {
			time.Sleep(tick)                   // wait the specified duration
			second, err = getRed(url, verbose) // get a new value
			if err != nil {
				if crash {
					panic(err)
				}
				log.Fatalf("redstat: fatal error, halting. Message was %q\n", err)
			}
			if verbose {
				log.Printf("Subsequent sample %d was %s\n", i, first.String())
			}
			difference = second.Subtract(first)
			difference.Duration = tick // set the requested duration
			report(difference, json)   // and report it
			first = second
			// check for ^C here
			if i == count-1 {
				return difference // just for testing
			}
		}
	}
	return difference // last one, for testing
}

// report produces human-oriented or json output
func report(r *r.Red, json bool) {
	if json {
		s, _ := r.MarshalJSON() // correct by construction
		fmt.Printf("red = %s", s)
	} else {
		fmt.Printf("red = %s\n", r.String())
	}
}

// getRed gets a datum, stopping or panicking on error
func getRed(url string, verbose bool) (*r.Red, error) {
	var red, zero *r.Red

	zero = r.Start()
	resp, err := http.Get(url)
	if err != nil {
		// This common case needs work, specifically an 'error.Is()' expression
		if strings.Contains(err.Error(), "connection refused") {
			// log.Printf("error type was %T\n", err)
			return zero, fmt.Errorf("getRed: connection refused, %q", err)
		}
		return zero, fmt.Errorf("getRed: get request failed, reported %q, %#v", err, err)
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Printf("in getRed, deferred resp.Body.Close() failed, ignored. Message was %q\n", err)
		}
	}()

	if verbose {
		// Look at the data we base averything on
		log.Printf("request = %#v\n", resp.Request)
		log.Printf("resp = %#v\n", resp)
		log.Printf("body = %q\n", resp.Body)
	}
	if resp.StatusCode > 299 {
		// XXX return a distinguished error, in case we want to handle it by skipping
		return zero, fmt.Errorf("getRed, read of body reported an error, with status code: %d and body: %s",
			resp.StatusCode, resp.Body)
	}

	red, err = redFromReader(resp.Body)
	if err != nil {
		return zero, err
	}
	return red, nil
}

func redFromReader(reader io.Reader) (*r.Red, error) {
	var line string
	var duration float64
	var red, zero r.Red

	x, err := ioutil.ReadAll(reader)
	// FIXME: ReadAll is really in io in go 1.16, not ioutil, but lints reject that.
	// Waiting for staticcheck and errcheck to be fixed
	if err != nil {
		return &zero, fmt.Errorf("io.Readall failed, reported %#v", err)
	}

	line = string(x)
	n, err := fmt.Sscanf(line, "%d, %d, %g", &red.Requests, &red.Errors, &duration)
	if err != nil {
		return &zero, fmt.Errorf("sscanf failed to scan a Red from %q, reported %#v", line, err)
	}
	if n < 3 {
		return &zero, fmt.Errorf("sscanf failed, read %d fields from %q", n, line)
	}
	red.Duration = time.Duration(duration) * time.Second
	return &red, nil
}
