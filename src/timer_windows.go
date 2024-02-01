package main

import (
	"syscall"
	"unsafe"
)

var (
	winmmDLL                      = syscall.NewLazyDLL("winmm.dll")
	procTimeBeginPeriod           = winmmDLL.NewProc("timeBeginPeriod")
	kernel32DLL                   = syscall.NewLazyDLL("kernel32.dll")
	procQueryPerformanceCounter   = kernel32DLL.NewProc("QueryPerformanceCounter")
	procQueryPerformanceFrequency = kernel32DLL.NewProc("QueryPerformanceFrequency")
)

func set_timer_resolution(period int) {
	procTimeBeginPeriod.Call(uintptr(period))
}

/*
On Windows the time functions can only give us a precision of 1ms.
The following functions are implemented using QPC instead, giving
similar precision to Unix.
*/
func get_timer() int64 {
	var res int64
	procQueryPerformanceCounter.Call(uintptr(unsafe.Pointer(&res)))
	return int64(res)
}

func timer_elapsed(start int64) int64 {
	var freq int64
	procQueryPerformanceFrequency.Call(uintptr(unsafe.Pointer(&freq)))

	elapsed := get_timer() - start
	elapsed *= 1000000000
	elapsed /= int64(freq)
	return elapsed
}
