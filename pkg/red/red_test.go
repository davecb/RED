package red

// red_test is GoConvey tests of REQUESTS, Errors and Durations

import (
	. "github.com/smartystreets/goconvey/convey"
	"log"
	"sync"
	"testing"
	"time"
)

// BenchmarkAdd is a timing test of the standard implementation
func BenchmarkAdd(b *testing.B) {
	var r = Start()
	var wg sync.WaitGroup

	wg.Add(b.N)
	Nanosleep = true
	for i := 0; i < b.N; i++ {
		go func() {
			r.Add(REQUESTS, 1)
			wg.Done()
		}()
	}
	wg.Wait()
	b.Logf("red reported %q\n", r.Now().String())
}

// BenchmarkBadAdd is a timing test of an implementation using locks. It and the
// channel-only implementation are both so fast that it's hard to get them to contend.
func BenchmarkBadAdd(b *testing.B) {
	var r = Start()
	var wg sync.WaitGroup

	wg.Add(b.N)
	Nanosleep = true
	for i := 0; i < b.N; i++ {
		go func() {
			r.BadAdd(REQUESTS, 1)
			wg.Done()
		}()
	}
	wg.Wait()
	b.Logf("red reported %q\n", r.Now().String())
}

// TestRedHappyPath confirms we're doing the operations properly
func TestRedHappyPath(t *testing.T) {
	var zero = &Red{}
	var r = Start()

	Convey("Given an initialized red ", t, func() {

		Convey("initially it should be (logically) zero", func() {
			//t.logf("r.String() %q, should equal logical zero\n", r.String())
			So(r.String(), ShouldEqual, zero.String())
		})

		Convey("When incremented, it increases by one", func() {
			_ = r.Add(REQUESTS, 1)
			_ = r.GetAll()
			//t.Logf("after GetAll in red_test, the value of red is now %#v\n", r) // It should change
			So(r.Requests, ShouldEqual, 1)
		})

		Convey("When set, it changed", func() {
			_ = r.Add(ERRORS, 11)
			_ = r.GetAll()
			So(r.Errors, ShouldEqual, 11)
		})

		Convey("When Now() is called, it sets Duration", func() {
			_ = r.Now()
			//t.Logf("red_test r.Duration = %#v\n", r.Duration)
			So(r.Duration, ShouldNotEqual, 0)
		})
	})
}

// TestRedUninitializedPath confirms we're doing the operations properly
func TestRedUninitializedPath(t *testing.T) {
	var r *Red

	Convey("Given the declaration of a red ", t, func() {
		Convey("Initially it should be nil", func() {
			So(r, ShouldEqual, nil)

		})
		Convey("Adding to nil should return an error", func() {
			// true of other operations, too
			err := r.Add(ERRORS, 1)
			//t.Logf("red_test err %q, should not be a nil\n", err)
			So(err.Error(), ShouldEqual, "r is nil, please call Start() first")
		})
		Convey("Calling Now() on it should work, however", func() {
			// Now() returns a Red instead of error, so it works around the error
			// so it can be used in the idiom log.Printf("%s\n", red.End().String())
			So(r.Now().String(), ShouldNotEqual, "r is nil, please call Start() first")
		})
		Convey("Turning it into a string should return an error message", func() {
			So(r.String(), ShouldEqual, "r is nil, please call Start() first")
		})
	})
}

// TestRedUnhappyPath confirms we're doing the operations properly
func TestRedUnhappyPath(t *testing.T) {
	var r = Start()

	Convey("Given an initialized red ", t, func() {

		Convey("After initialization it should be (logical) zero", func() {
			//t.Logf("red_test r.String() %q, should equal logical zero\n", r.String())
			So(r.String(), ShouldEqual, "0, 0, 0.000000s")
		})
		//SkipConvey("Passing bad operations to send should panic worker", func(c C) {
		//	// the panic happens in worker(), not send,
		//	// FIXME, in future, see if goConvey can pass it back to here
		//	c.So(func() { r.send(-1, ERRORS, 1) }, ShouldPanic)
		//
		//})
		//SkipConvey("Passing bad operands to send should panic worker", func(c C) {
		//	// the panic happens in worker(), not send
		//	c.So(func() { r.send(set, NONE, 1) }, ShouldPanic)
		//})
	})
}

// ExampleRed show how to use it, letting Now() compute the Duration
func ExampleRed() {
	// Initialize a Red
	var r = Start()

	// Do some operations, let's say 3
	for i := 0; i < 3; i++ {
		time.Sleep(1 * time.Second)
		// Update Red each time
		r.Add(REQUESTS, 1)
		if i == 1 {
			r.Add(ERRORS, 1)
		}
		r.Now()
		log.Printf("%s\n", r.String())
	}
	j, err := r.MarshalJSON()
	if err != nil {
		panic(err)
	}
	log.Printf("%s\n", j)
	// Output:
}
