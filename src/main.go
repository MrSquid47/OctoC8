package main

import (
	"fmt"
	"log"
	"math"
	"runtime"
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	mathp "github.com/golangplus/math"
)

const RENDER_MULTIPLIER = 20
const CPU_FREQUENCY = 600
const QUIRK_DISPLAY_WAIT = false

var ticks int64

var draw_wait bool = false
var last_cycle int64

var fullscreen_state = false
var window_state []int = make([]int, 4)

func init() {
	// This is needed to arrange that main() runs on main thread.
	// See documentation for functions that are only allowed to be called from the main thread.
	runtime.LockOSThread()
}

func main() {
	set_timer_resolution(1)

	beep_init()

	err := glfw.Init()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	fmt.Println("initialized")
	mem_init()
	load_file("../roms/snake.ch8")
	//load_file("../roms/chip8-test-suite.ch8")
	//load_file("../roms/BRIX")

	//load_file("../roms/test-suite/1-chip8-logo.ch8")
	//load_file("../roms/test-suite/2-ibm-logo.ch8")
	//load_file("../roms/test-suite/3-corax+.ch8")
	//load_file("../roms/test-suite/4-flags.ch8")
	//load_file("../roms/test-suite/5-quirks.ch8")
	//load_file("../roms/test-suite/6-keypad.ch8")
	//load_file("../roms/test-suite/7-beep.ch8")
	//load_file("../roms/test-suite/8-scrolling.ch8")

	window, err := glfw.CreateWindow(64*RENDER_MULTIPLIER, 32*RENDER_MULTIPLIER, "OctoC8", nil, nil)
	if err != nil {
		panic(err)
	}
	window.SetKeyCallback(key_callback)

	window.MakeContextCurrent()
	// Enable VSync
	glfw.SwapInterval(1)

	if err := gl.Init(); err != nil {
		log.Fatalln(err)
	}

	var fboID uint32 = 0
	var textureID uint32
	gl.GenTextures(1, &textureID)
	gl.GenFramebuffers(1, &fboID)

	// Timers
	go func() {
		var timers = [...]*timer{&timer_delay, &timer_sound}
		for i := range timers {
			timers[i].prevtick = get_timer()
		}

		for {
			for i := range timers {
				if timer_elapsed(timers[i].prevtick) >= 16666667 {
					if timers[i].value > 0 {
						//fmt.Printf("timer: 0x%X\n", timers[i].value)
						timers[i].value--
						//fmt.Printf("timer: 0x%X\n", timers[i].value)
					}

					if timers[i] == &timer_delay {
						draw_wait = false
					}

					timers[i].prevtick = get_timer()

					if timers[i] == &timer_sound && timers[i].value == 0 {
						beep_stop()
					}
				}
			}

			if (16666667 - timer_elapsed(timer_delay.prevtick)) >= 1000000 {
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()

	// CPU Loop
	go func() {
		var cycsteps int = int(math.Round(CPU_FREQUENCY / 200.0))
		var substeps int
		last_cycle = get_timer()

		for {
			cpu_step()
			ticks++
			substeps++
			keywait = -1

			if CPU_FREQUENCY >= 100 {
				if substeps >= cycsteps {
					substeps = 0
					for {
						if timer_elapsed(last_cycle) >= 5000000 {
							last_cycle = get_timer()
							break
						} else if (5000000 - timer_elapsed(last_cycle)) > 1000000 {
							// Only busy wait when we drop under 1ms remaining
							time.Sleep(1 * time.Millisecond)
						}
					}
				}
			} else {
				time.Sleep(time.Duration(math.Round(1000.0/CPU_FREQUENCY)) * time.Millisecond)
			}
		}
	}()

	go func() {
		startt := time.Now().UnixNano()
		for time.Now().UnixNano() == startt {

		}

		fmt.Printf("diff: %d\n", time.Now().UnixNano()-startt)

		for {
			time.Sleep(1000 * time.Millisecond)
			fmt.Printf("ticks: %d\n", ticks)
			ticks = 0
		}
	}()

	// Main Render Loop:
	for !window.ShouldClose() {
		// Stop rendering so we don't crash when minimized
		if window.GetAttrib(glfw.Iconified) == 0 {
			//gl.ClearColor(0.1, 0.1, 0.1, 1.0)
			gl.Clear(gl.COLOR_BUFFER_BIT)

			var texture_data []byte = make([]byte, 64*32*4)
			for i := range video_memory {
				if video_memory[i] {
					texture_data[i*4] = 255
					texture_data[i*4+1] = 255
					texture_data[i*4+2] = 255
					texture_data[i*4+3] = 255
				} else {
					texture_data[i*4] = 10
					texture_data[i*4+1] = 10
					texture_data[i*4+2] = 10
					texture_data[i*4+3] = 255
				}
			}

			var texture_data_flipped []byte = make([]byte, 64*32*4)
			for i := 0; i < (32); i++ {
				for j := 0; j < (64*4)-1; j++ {
					line_mult := 64 * 4 * (31 - i)
					texture_data_flipped[line_mult+j] = texture_data[(64*4*i)+j]
				}
			}

			gl.BindTexture(gl.TEXTURE_2D, textureID)
			gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 64, 32, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(texture_data_flipped))

			gl.BindFramebuffer(gl.READ_FRAMEBUFFER, fboID)
			gl.FramebufferTexture2D(gl.READ_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, textureID, 0)
			gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, 0) // if not already bound

			ww, wh := window.GetSize()
			var origRatio float64 = 64.0 / 32.0
			var targRatio float64 = float64(ww) / float64(wh)
			var scaledWidth = ww
			var scaledHeight = wh

			if origRatio > targRatio {
				scaledHeight = int(float64(ww) / origRatio)
			} else if origRatio < targRatio {
				scaledWidth = int(float64(wh) * origRatio)
			}

			gl.BlitFramebuffer(0, 0, 64, 32, int32(float64(ww-scaledWidth)/origRatio), int32(float64(wh-scaledHeight)/origRatio), int32(scaledWidth)+int32(float64(ww-scaledWidth)/origRatio), int32(scaledHeight)+int32(float64(wh-scaledHeight)/origRatio), gl.COLOR_BUFFER_BIT, gl.NEAREST)

			window.SwapBuffers()
		}

		glfw.PollEvents()
	}
}

func key_callback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	// shortcuts
	if key == glfw.KeyEnter && action == glfw.Press {
		if w.GetKey(glfw.KeyLeftAlt) == glfw.Press || w.GetKey(glfw.KeyRightAlt) == glfw.Press {
			set_fullscreen(w, !fullscreen_state)
		}
	}

	// Chip8 key handling
	var kp int
	switch key {
	case glfw.Key1:
		kp = 0x1
	case glfw.Key2:
		kp = 0x2
	case glfw.Key3:
		kp = 0x3
	case glfw.Key4:
		kp = 0xC
	case glfw.KeyQ:
		kp = 0x4
	case glfw.KeyW:
		kp = 0x5
	case glfw.KeyE:
		kp = 0x6
	case glfw.KeyR:
		kp = 0xD
	case glfw.KeyA:
		kp = 0x7
	case glfw.KeyS:
		kp = 0x8
	case glfw.KeyD:
		kp = 0x9
	case glfw.KeyF:
		kp = 0xE
	case glfw.KeyZ:
		kp = 0xA
	case glfw.KeyX:
		kp = 0x0
	case glfw.KeyC:
		kp = 0xB
	case glfw.KeyV:
		kp = 0xF
	default:
		return
	}

	if action == glfw.Release {
		keypad[kp] = false
		keywait = kp
	} else {
		keypad[kp] = true
	}
}

func set_fullscreen(window *glfw.Window, set bool) {
	if set == fullscreen_state {
		return
	}

	if !fullscreen_state {
		mon := get_window_monitor(window)
		window_state[0], window_state[1] = window.GetPos()
		window_state[2], window_state[3] = window.GetSize()
		window.SetMonitor(mon, 0, 0, mon.GetVideoMode().Width, mon.GetVideoMode().Height, mon.GetVideoMode().RefreshRate)
		window.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
		fullscreen_state = true
	} else {
		window.SetMonitor(nil, window_state[0], window_state[1], window_state[2], window_state[3], 0)
		window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
		fullscreen_state = false
	}

	glfw.SwapInterval(1)
}

func get_window_monitor(window *glfw.Window) *glfw.Monitor {
	var wx, wy, ww, wh int
	var mx, my, mw, mh int
	var overlap, bestoverlap int
	var bestmonitor *glfw.Monitor
	var monitors []*glfw.Monitor
	var mode *glfw.VidMode

	bestoverlap = 0
	bestmonitor = nil

	wx, wy = window.GetPos()
	ww, wh = window.GetSize()
	monitors = glfw.GetMonitors()

	for i := 0; i < len(monitors); i++ {
		mode = monitors[i].GetVideoMode()
		mx, my = monitors[i].GetPos()
		mw = mode.Width
		mh = mode.Height

		overlap = mathp.MaxI(0, mathp.MinI(wx+ww, mx+mw)-mathp.MaxI(wx, mx)) * mathp.MaxI(0, mathp.MinI(wy+wh, my+mh)-mathp.MaxI(wy, my))

		if bestoverlap < overlap {
			bestoverlap = overlap
			bestmonitor = monitors[i]
		}
	}

	return bestmonitor
}
