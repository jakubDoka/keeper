package util

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync/atomic"
)

// BodyToReader reads the body of request and returns meaningfull error
// that does not need modifications.
func BodyToReader(r *http.Request) (Reader, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return Reader{}, WrapErr("failed to read request body", err)
	}

	return NewReader(body), nil
}

// WrapErr wraps error with message (message: error)
func WrapErr(message string, err error) error {
	return fmt.Errorf(message+": %s", err)
}

type AtomicInt32 struct {
	value int32
}

func (a *AtomicInt32) Set(value int32) {
	atomic.StoreInt32(&a.value, value)
}

func (a *AtomicInt32) Get() int32 {
	return atomic.LoadInt32(&a.value)
}

func CheckSysCallError(err error, value string) bool {
	switch e := err.(type) {
	case *net.OpError:
		if e, ok := e.Err.(*os.SyscallError); ok {
			fmt.Println(e.Syscall)
			return e.Syscall == value
		}
	}
	return false
}

func Clamp(v, min, max uint32) uint32 {
	if v > max {
		return max
	}
	if v < min {
		return min
	}
	return v
}
