package main

import "sync/atomic"

//AtomicInt32 to prevent direct get/set
type AtomicInt32 struct {
	*int32
}

//AtomicInt64 to prevent direct get/set
type AtomicInt64 struct {
	*int64
}

func (a AtomicInt64) set(n int64) {
	atomic.StoreInt64(a.int64, n)
}

func (a AtomicInt64) get() int64 {
	return atomic.LoadInt64(a.int64)
}

func (a AtomicInt64) add(n int64) int64 {
	return atomic.AddInt64(a.int64, n)
}

func (a AtomicInt32) set(n int32) {
	atomic.StoreInt32(a.int32, n)
}

func (a AtomicInt32) get() int32 {
	return atomic.LoadInt32(a.int32)
}

func (a AtomicInt32) add(n int32) int32 {
	return atomic.AddInt32(a.int32, n)
}
