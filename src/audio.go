package main

import (
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/generators"
	"github.com/faiface/beep/speaker"
)

var beeping bool = false
var prevbeep int64
var sr beep.SampleRate

func beep_init() {
	sr = beep.SampleRate(44100)
	speaker.Init(sr, 1800)
}

func beep_start() {
	if !beeping {
		sine, _ := generators.SquareTone(sr, 300)
		volume := &effects.Volume{
			Streamer: sine,
			Base:     2,
			Volume:   -3,
			Silent:   false,
		}
		speaker.Play(volume)
		beeping = true
	}
}

func beep_stop() {
	if beeping && (time.Now().UnixMilli()-prevbeep) > 100 {
		speaker.Clear()
		beeping = false
	}
}
