//go:build !windows

package main

import "time"

// This is only relevant on windows.
func set_timer_resolution(period int) {}

func get_timer() int64 {
	return int64(time.Now().UnixNano())
}

func timer_elapsed(start int64) int64 {
	return time.Now().UnixNano() - start
}
