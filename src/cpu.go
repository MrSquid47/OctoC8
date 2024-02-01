package main

import (
	"fmt"
	"math/rand"
	"time"
)

var reg_V0,
	reg_V1,
	reg_V2,
	reg_V3,
	reg_V4,
	reg_V5,
	reg_V6,
	reg_V7,
	reg_V8,
	reg_V9,
	reg_VA,
	reg_VB,
	reg_VC,
	reg_VD,
	reg_VE,
	reg_VF byte

var reg_I uint16
var PC int = 0x200

var stack []int = make([]int, 256)
var stack_ptr uint = 0

type timer struct {
	value    uint8
	prevtick int64
}

var timer_delay timer
var timer_sound timer

var draw_sync bool

func cpu_step() {
	instr := uint16(sys_memory[PC])<<8 | uint16(sys_memory[PC+1])

	// increment the PC as a default case; may be modified later
	PC = PC + 2

	// (instr>>(8*0))&0xff) //2nd byte
	//fmt.Printf("%b\n", (instr))
	//fmt.Printf("0x%X: 0x%X\n", PC-2, (instr)&0xFFFF)
	//fmt.Printf("%X\n", (instr>>8)&0x0FFF)

	// Decode & Execute
	switch (instr >> (12)) & 0xf {
	case 0x0:
		switch (instr >> (8)) & 0xf {
		case 0x0:
			switch (instr >> (0)) & 0xff {
			case 0xE0: // clear screen
				for i := range video_memory {
					video_memory[i] = false
				}
			case 0xEE:
				stack_ptr--
				PC = int(stack[stack_ptr])
			}
		default:
			// unimplemented
		}
	case 0x1: // jump
		PC = int(instr & 0xFFF)
	case 0x2:
		stack[stack_ptr] = PC
		stack_ptr++
		PC = int(instr & 0xFFF)
	case 0x3:
		reg := fetch_register(uint8((instr >> (8)) & 0xf))
		if uint16(*reg) == (instr>>(0))&0xff {
			PC = PC + 2
		}
	case 0x4:
		reg := fetch_register(uint8((instr >> (8)) & 0xF))
		if *reg != byte((instr & 0xFF)) {
			PC = PC + 2
		}
	case 0x5:
		reg_X := fetch_register(uint8((instr >> (8)) & 0xf))
		reg_Y := fetch_register(uint8((instr >> (4)) & 0xf))
		if *reg_X == *reg_Y {
			PC = PC + 2
		}
	case 0x6:
		reg := fetch_register(uint8((instr >> (8)) & 0xf))
		*reg = uint8((instr) & 0xFF)
	case 0x7:
		reg := fetch_register(uint8((instr >> (8)) & 0xf))
		*reg = *reg + uint8((instr)&0xFF)
		//fmt.Printf("ADD V%X = 0x%X\n", int((instr>>(8))&0xf), *reg)
	case 0x8:
		reg_X := fetch_register(uint8((instr >> (8)) & 0xf))
		reg_Y := fetch_register(uint8((instr >> (4)) & 0xf))
		switch instr & 0xF {
		case 0x0:
			*reg_X = *reg_Y
		case 0x1:
			*reg_X = *reg_X | *reg_Y
			reg_VF = 0
		case 0x2:
			*reg_X = *reg_X & *reg_Y
			reg_VF = 0
		case 0x3:
			*reg_X = *reg_X ^ *reg_Y
			reg_VF = 0
		case 0x4:
			var vfcp byte = 0
			if int32(*reg_X)+int32(*reg_Y) > 255 {
				vfcp = 1
			}
			*reg_X = *reg_X + *reg_Y
			reg_VF = vfcp
		case 0x5:
			var vfcp byte = 1
			if *reg_X < *reg_Y {
				vfcp = 0
			}
			*reg_X = *reg_X - *reg_Y
			reg_VF = vfcp
		case 0x6:
			*reg_X = *reg_Y
			vfcp := *reg_X & 0x1
			*reg_X = *reg_X >> 1
			reg_VF = vfcp
		case 0x7:
			var vfcp byte = 1
			if *reg_Y < *reg_X {
				vfcp = 0
			}
			*reg_X = *reg_Y - *reg_X
			reg_VF = vfcp
		case 0xE:
			*reg_X = *reg_Y
			vfcp := *reg_X >> 7
			*reg_X = *reg_X << 1
			reg_VF = vfcp
		default:
			fmt.Printf("Unimplemented instruction: 0x%X\n", instr)
		}
	case 0x9:
		reg_X := fetch_register(uint8((instr >> (8)) & 0xf))
		reg_Y := fetch_register(uint8((instr >> (4)) & 0xf))
		if *reg_X != *reg_Y {
			PC = PC + 2
		}
	case 0xA:
		reg_I = uint16(instr & 0xFFF)
	case 0xB:
		PC = int(instr&0xFFF) + int(reg_V0)
	case 0xC:
		reg := fetch_register(uint8((instr >> (8)) & 0xf))
		mask := byte(instr & 0xFF)
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
		*reg = byte(r1.Intn(256)) & mask
	case 0xD:
		if !draw_sync {
			draw_sync = true
			draw_wait = true
			PC = PC - 2
			break
		} else {
			if draw_wait {
				PC = PC - 2
				break
			}
		}
		draw_sync = false

		reg_X := fetch_register(uint8((instr >> (8)) & 0xf))
		reg_Y := fetch_register(uint8((instr >> (4)) & 0xf))
		var coord_X byte
		coord_Y := *reg_Y & 31
		reg_VF = 0x0
		var i uint16
		for i = 0; i < (instr & 0xf); i++ {
			if coord_Y >= 32 {
				break
			}

			var sprite_row byte = sys_memory[reg_I+i]
			coord_X = *reg_X & 63
			for j := 0; j < 8; j++ {
				if coord_X >= 64 {
					break
				}

				var pixel bool = (sprite_row & (1 << (7 - j))) != 0
				coord_loc := get_coord_location(int(coord_X), int(coord_Y))
				if pixel {
					if video_memory[coord_loc] {
						reg_VF = 0x1
					}
					video_memory[coord_loc] = !video_memory[coord_loc]
				}
				coord_X++
			}
			coord_Y++
		}
	case 0xE:
		reg := fetch_register(uint8((instr >> (8)) & 0xf))
		switch (instr >> (0)) & 0xff {
		case 0x9E:
			if keypad[*reg&0xf] {
				PC += 2
			}
		case 0xA1:
			if !keypad[*reg&0xf] {
				PC += 2
			}
		}
	case 0xF:
		reg := fetch_register(uint8((instr >> (8)) & 0xf))
		switch (instr >> (0)) & 0xff {
		case 0x07:
			*reg = timer_delay.value
		case 0x0A:
			if keywait != -1 {
				*reg = byte(keywait)
			} else {
				PC = PC - 2
			}
		case 0x15:
			timer_delay.value = *reg
		case 0x18:
			if timer_sound.value == 0 && *reg != 0 {
				beep_start()
				prevbeep = time.Now().UnixMilli()
			}
			timer_sound.value = *reg
		case 0x1E:
			reg_I = reg_I + uint16(*reg)
		case 0x29:
			char := (*reg & 0xf)
			reg_I = uint16(char*5 + 80)
		case 0x33:
			reg := fetch_register(uint8((instr >> (8)) & 0xf))
			sys_memory[reg_I] = *reg / 100
			sys_memory[reg_I+1] = (*reg / 10) % 10
			sys_memory[reg_I+2] = *reg % 10
		case 0x55:
			for i := 0; i <= int(instr>>(8)&0xf); i++ {
				sys_memory[int(reg_I)+i] = *fetch_register(uint8(i))
			}
			reg_I = reg_I + (instr >> (8) & 0xf) + 1
		case 0x65:
			for i := 0; i <= int(instr>>(8)&0xf); i++ {
				*fetch_register(uint8(i)) = sys_memory[int(reg_I)+i]
			}
			reg_I = reg_I + (instr >> (8) & 0xf) + 1
		}
	default:
		fmt.Printf("Unknown instruction: 0x%X\n", instr)
	}
}

func fetch_register(regnum uint8) *byte {
	switch regnum {
	case 0x0:
		return &reg_V0
	case 0x1:
		return &reg_V1
	case 0x2:
		return &reg_V2
	case 0x3:
		return &reg_V3
	case 0x4:
		return &reg_V4
	case 0x5:
		return &reg_V5
	case 0x6:
		return &reg_V6
	case 0x7:
		return &reg_V7
	case 0x8:
		return &reg_V8
	case 0x9:
		return &reg_V9
	case 0xA:
		return &reg_VA
	case 0xB:
		return &reg_VB
	case 0xC:
		return &reg_VC
	case 0xD:
		return &reg_VD
	case 0xE:
		return &reg_VE
	case 0xF:
		return &reg_VF
	}
	fmt.Printf("Invalid register: V%X\n", regnum)
	return nil
}
