package red

// Red is the minimum set of metrics we commonly use.
// The public members are for callers, everything
// after init() is the back-end implementation.
// This uses channels to avoid locking, w preformance
// problem we have elsewhere.

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Red is the master struct for tracking REQUESTS, Errors and Duration
type Red struct {
	Requests  int64         `json:"requests"`
	Errors    int64         `json:"errors"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
}

// RED is the minimum signature of a Red implementation
type RED interface {
	Start() *Red
	Add(string, int64) error
	Now() *Red
	String() string
}

// Fields are an enum of the fields we want to update
type Fields int

const (
	// REQUESTS is a request to update the Requests field
	REQUESTS Fields = iota
	// ERRORS is one to update the Error field
	ERRORS
	// DURATION should not be updated by callers
	DURATION
	// and NONE means "no fields"
	NONE
)

func (field Fields) String() string {
	switch field {
	case REQUESTS:
		return "REQUESTS"
	case ERRORS:
		return "ERRORS"
	case DURATION:
		return "DURATION"
	default:
		return "unknown-field"
	}
}

// Start returns a handle and sets the Start time.
// Calling it repeatedly merely causes it to restart the counts.
func Start() *Red {
	// Use "main" so that it will work the very first time it's called
	main.send(start, NONE, 0)
	// The user interface strictly uses this copy, so that code can't actually
	// touch main concurrently with the worker code.
	var ui = &Red{
		0,
		0,
		0,
		time.Now(),
	}
	return ui
}

// Add sends an add message to a singleton worker
func (r *Red) Add(f Fields, val int64) error {
	if r == nil {
		return fmt.Errorf("r is nil, please call Start() first")
	}
	switch f {
	case REQUESTS, ERRORS:
		if Nanosleep {
			// For benchmarking only, this makes the program just slow
			// enough that it can experience contention. It's too darned fast!
			time.Sleep(100 * time.Nanosecond)
			// if you want contention from Add, you need to use 1,000,000 microsecond, 1 second
			//time.Sleep(1000000 * time.Microsecond)
		}
		r.send(add, f, val)
		tmp := r.receive()
		r.Requests, r.Errors, r.Duration = tmp.Requests, tmp.Errors, tmp.Duration
		return nil
	default:
		return fmt.Errorf("usage error, unsupported operand %q for Add(%q, %d)", f.String(), f, val)
	}
}

// BadAdd applies a lock before sending, for benchmarking only.
func (r *Red) BadAdd(f Fields, val int64) error {
	mu.Lock()
	defer mu.Unlock()

	if r == nil {
		return fmt.Errorf("r is nil, please call Start() first")
	}
	switch f {
	case REQUESTS, ERRORS:
		if Nanosleep {
			// For benchmarking only: see note in Add, above.
			// Without this, there is no lock contention, and
			// BadAdd is often faster than Add.  And yes, that
			// means we could have got away with using locks.
			time.Sleep(100 * time.Nanosecond)
		}
		r.send(add, f, val)
		tmp := r.receive()
		r.Requests, r.Errors, r.Duration = tmp.Requests, tmp.Errors, tmp.Duration
		return nil
	default:
		return fmt.Errorf("usage error, unsupported operand %q for Add(%q, %d)", f.String(), f, val)
	}
}

// mu and Nanosleep are used by BadAdd only
var mu sync.Mutex
var Nanosleep = false

// Set sends a set message
func (r *Red) Set(f Fields, val int64) error {
	if r == nil {
		return fmt.Errorf("r is nil, please call Start() first")
	}
	switch f {
	case REQUESTS, ERRORS:
		r.send(set, f, val)
		tmp := r.receive()
		r.Requests, r.Errors, r.Duration = tmp.Requests, tmp.Errors, tmp.Duration
		return nil
	default:
		return fmt.Errorf("usage error, unsupported operand %q for Set(%q, %d)", f.String(), f, val)
	}
}

// GetAll sends a getall message
func (r *Red) GetAll() error {
	if r == nil {
		return fmt.Errorf("r is nil, please call Start() first")
	}
	r.send(getall, NONE, 0)
	tmp := r.receive()
	r.Requests, r.Errors, r.Duration = tmp.Requests, tmp.Errors, tmp.Duration
	return nil
}

// Now fills in the Duration field of a Red. Often used to end a time-period,
// as in fmt.Printf("%s\n", red.Now().String()).
// It will compute Red.Duration each time it's called, using time.Since()
func (r *Red) Now() *Red {
	if r == nil {
		// create a temporary one so we don't have to return a non-Red
		r = &Red{}
	}
	r.send(now, NONE, 0)
	tmp := r.receive()
	r.Requests, r.Errors, r.Duration = tmp.Requests, tmp.Errors, tmp.Duration
	return r
}

// String converts r into a string.  If you want it to contain anything,
// interesting, call Start() and Now() before calling String()
func (r *Red) String() string {
	if r == nil {
		return "r is nil, please call Start() first"
	}
	return fmt.Sprintf("%d, %d, %fs", r.Requests, r.Errors, r.Duration.Seconds())
}

// MarshalJSON converts r into a shortened json. As with String(),
// call Now() first if you want to know the Duration.
func (r *Red) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Requests int64         `json:"requests"`
		Errors   int64         `json:"errors"`
		Duration time.Duration `json:"duration"`
	}{
		r.Requests,
		r.Errors,
		r.Duration,
	})
}

// Subtract is a convenience function for a caller who subtracts two measurements to get
// a rate, a common use case.
func (r *Red) Subtract(v *Red) *Red {
	r.Requests -= v.Requests
	r.Errors -= v.Errors
	r.Duration -= v.Duration
	return r
}

// Private members of Red
// main is the internal Red variable, protected from concurrent access
var main Red
var toWorker chan msg
var fromWorker chan *Red
var verbose = false

// init creates a back end
func init() {
	// The channel size is a tuning parameter, and should
	// be larger than the number of callers so the callers
	// don't have to wait while the goroutine make the
	// changes single-threaded. 1 is too low, while
	// 1000 is only slightly better than 100 in MY benchmark.
	// YOUR milage will vary.
	toWorker = make(chan msg, 100)
	fromWorker = make(chan *Red, 100)
	go worker()
}

// msg is what the UI sends to the worker via a channel
type msg struct {
	operation ops    // add, getall, set, etc
	operand   Fields // request, error and Duration
	value     int64  // its value
}

// ops is an enum of the operations that the package does
type ops int

const (
	add ops = iota
	getall
	set
	start
	now
)

func (op ops) String() string {
	switch op {
	case add:
		return "add"
	case getall:
		return "getall"
	case set:
		return "set"
	case start:
		return "StartTime"
	case now:
		return "now"
	}
	return "unknown operation"
}

// send sends a request to the worker from the UI
func (r *Red) send(operation ops, operand Fields, value int64) {
	toWorker <- msg{
		operation,
		operand,
		value,
	}
}

// receive gets stuff sent back from worker to the UI
func (r *Red) receive() *Red {
	msg := <-fromWorker
	return msg
}

// reply sends to fromWorker, for worker to use to reply to the UI
// note that main doesn't get the error, that's specific to the
// call from the UI
func (r *Red) reply(s Red) {
	var tmp = s
	fromWorker <- &tmp
}

// Worker serializes the senders, manipulates main.
func worker() {
	var tmp Red

	for m := range toWorker {
		if verbose {
			log.Printf("worker got %q, %q, %d\n", m.operation.String(), m.operand, m.value)
		}
		switch m.operation {
		case start:
			// StartTime a time period
			main = Red{
				0,
				0,
				0,
				time.Now(),
			}

		case add:
			// add to a field
			switch m.operand {
			case REQUESTS:
				main.Requests += m.value
			case ERRORS:
				main.Errors += m.value
			default:
				panic(fmt.Errorf("programmer error, unknown operand %q in %#v", m.operand, m))
			}
			main.reply(main)

		case set:
			// override a field
			switch m.operand {
			case REQUESTS:
				main.Requests = m.value
			case ERRORS:
				main.Errors = m.value
			default:
				panic(fmt.Errorf("programmer error, unknown operand %q in %#v", m.operand, m))
			}
			main.reply(main)

		case getall:
			main.reply(main)

		case now:
			// report the values, as of now. Doesn't touch main
			tmp = main
			tmp.Duration = time.Since(main.StartTime)
			main.reply(tmp)
		default:
			panic(fmt.Errorf("programmer error, unknown opcode %q in %#v", m.operation.String(), m))
		}
		if verbose {
			log.Printf("after that operation, main = %#v\n", main)
		}
	}
}
