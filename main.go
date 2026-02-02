package main

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	width  = 44
	height = 44
)

var (
	colors = []string{
		"\x1b[31m", // red
		"\x1b[32m", // green
		"\x1b[33m", // yellow
		"\x1b[34m", // blue
		"\x1b[35m", // magenta
		"\x1b[36m", // cyan
	}

	grid [height][width]int
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// init grid
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			grid[y][x] = rand.Intn(10)
		}
	}

	for {
		clearScreen()
		updateGrid()
		renderGrid()
		time.Sleep(40 * time.Millisecond) // faster turbo mode
	}
}

func clearScreen() {
	fmt.Print("\x1b[2J\x1b[H")
}

func updateGrid() {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {

			// tiny random drift
			grid[y][x] += rand.Intn(3) - 1

			// neighbor influence (very cheap)
			if y > 0 {
				grid[y][x] += grid[y-1][x] / 10
			}
			if x > 0 {
				grid[y][x] += grid[y][x-1] / 10
			}

			// clamp
			if grid[y][x] < 0 {
				grid[y][x] = 0
			}
			if grid[y][x] > 20 {
				grid[y][x] = 20
			}
		}
	}
}

func renderGrid() {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {

			v := grid[y][x]

			char := energyChar(v)
			color := colors[v%len(colors)]

			fmt.Printf("%s%s", color, char)
		}
		fmt.Print("\x1b[0m\n")
	}
}

func energyChar(v int) string {
	switch {
	case v < 3:
		return " "
	case v < 6:
		return "∙"
	case v < 9:
		return "•"
	case v < 12:
		return "◉"
	case v < 15:
		return "◎"
	default:
		return "❖"
	}
}
