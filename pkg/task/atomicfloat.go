package task

import (
	"sync/atomic"
	"unsafe"
)

type AtomicFloat32 struct {
	value uint32
}

func NewF32(value float32) *AtomicFloat32 {
	f := &AtomicFloat32{}
	f.Set(value)
	return f
}

func (f *AtomicFloat32) Get() float32 {
	v := atomic.LoadUint32(&f.value)
	return *(*float32)(unsafe.Pointer(&v))
}

func (f *AtomicFloat32) Set(p float32) {
	v := *(*uint32)(unsafe.Pointer(&p))
	atomic.StoreUint32(&f.value, v)
}

type AtomicFloat64 struct {
	value uint64
}

func NewF64(value float64) *AtomicFloat64 {
	f := &AtomicFloat64{}
	f.Set(value)
	return f
}

// Returns the atomic float value
func (m *AtomicFloat64) Get() float64 {
	// Convert uint64 to float64 and return
	v := atomic.LoadUint64(&m.value)
	return *(*float64)(unsafe.Pointer(&v))
}

// Set the atomic float value
func (m *AtomicFloat64) Set(p float64) {
	// convert float64 to uint64 and store
	v := *(*uint64)(unsafe.Pointer(&p))
	atomic.StoreUint64(&m.value, v)
}
