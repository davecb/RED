package main

// red_test is GoConvey test of requests, errors and Durations obkects

import (
	"fmt"
	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.indexexchange.com/exchange-node/machine-learning/model-uploader/internal/red"
	"math/rand"
	"net/http"
	"testing"
)

var delay, count int
var url = "http://localhost:7723/metrics"
var verbose = false
var json = false
var crash = false
var zero red.Red

// TestEndToEnd tests handling of the url, delay and count parameters, using
// a real server that's just idling
func SkipTestEndToEnd(t *testing.T) {

	// test delay = 1 count = 2
	Convey("Given a URL and 10 2, redstat returns two reports", t, func() {
		delay = 1
		count = 2
		total := redstat(url, delay, count, verbose, json, crash)
		Convey("When redstat returns, red.requests will be 1 and red.duration will be non-zero ", func() {
			So(total.Duration, ShouldBeGreaterThan, zero.Duration)
			So(total.Requests, ShouldEqual, 0)
		})
	})

	// Test a delay of 10, count = 1
	Convey("Given a delay of 10 and a count of 1", t, func() {
		// test delay == 1 count = 1
		delay = 1
		count = 1
		total := redstat(url, delay, count, verbose, json, crash)
		Convey("When redstat returns, red.duration will be non-zero", func() {
			So(total.Duration, ShouldBeGreaterThan, zero.Duration)
		})
	})

	// Test a reachable but bad URL, tell it to panic so that we can detect that
	Convey("Given a bad URL, redstat should fail to scan a Red from it...", t, func() {
		url = "http://example.com:80"
		crash = true
		So(redstat(url, delay, count, verbose, json, crash), ShouldPanic)
	})

	// Test an unreachable URL, tell it to panic so that we can detect that
	Convey("Given an unreachable URL, redstat times out after 30 seconds and exits", t, func() {
		url = "http://example.com:5280"
		crash = true
		So(redstat(url, delay, count, verbose, json, crash), ShouldPanic)
	})

	Convey("Given just a URL, redstat returns data since program start", t, func() {
		// test delay == 0
		total := redstat(url, delay, count, verbose, json, crash)
		Convey("When redstat returns, red.duration will be non-zero", func() {
			So(total.Duration, ShouldBeGreaterThan, zero.Duration)
		})
	})

}

// TestMocked does tests using a fake http server
func TestMocked(t *testing.T) {

	Convey("Given just a URL, redstat returns a report since program start", t, func() {
		// Set up a mock http server
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET",
			url, httpmock.NewStringResponder(200, "0, 0, 42.0"))

		// test delay == 0, triggering a single report
		total := redstat(url, delay, count, verbose, json, crash)
		Convey("When redstat returns, red.duration will be 42.0", func() {
			So(total.Duration, ShouldBeGreaterThan, zero.Duration)
		})
	})

	// test delay = 10 count = 1
	Convey("Given a url and 1 1, redstat returns a single report", t, func() {
		// Set up a mock http server
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET",
			url, func(req *http.Request) (*http.Response, error) {
				// This will produce 18, then 87...
				return httpmock.NewStringResponse(250,
					fmt.Sprintf("%d, 0, 5280.0", rand.Intn(100))), nil
			})

		delay = 1
		count = 1
		total := redstat(url, delay, count, verbose, json, crash)
		Convey("When redstat returns, red.requests will be 6 and red.duration will be 10", func() {
			So(total.Duration.String(), ShouldEqual, "1s")
			So(total.Requests, ShouldEqual, 6)
		})
	})

}
