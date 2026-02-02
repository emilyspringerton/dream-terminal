package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

const (
	width  = 44
	height = 44
)

type Cell struct {
	value float64
	noise float64
}

var grid [height][width]Cell
var next [height][width]Cell

var colors = []string{
	"\x1b[31m", // red
	"\x1b[32m", // green
	"\x1b[33m", // yellow
	"\x1b[34m", // blue
	"\x1b[35m", // magenta
	"\x1b[36m", // cyan
}

var chars = []string{
	" ", "∙", "•", "◦", "◉", "◎", "●", "❖",
}

func main() {
	rand.Seed(time.Now().UnixNano())

	initGrid()

	for {
		clearScreen()
		stepSimulation()
		render()
		time.Sleep(50 * time.Millisecond)
	}
}

func initGrid() {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			grid[y][x].value = rand.Float64()
			grid[y][x].noise = rand.Float64() * 0.2
		}
	}
}

func stepSimulation() {

	// occasional shock to prevent equilibrium
	if rand.Float64() < 0.03 {
		injectShock()
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {

			sum := 0.0
			count := 0

			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					ny := y + dy
					nx := x + dx

					if ny >= 0 && ny < height && nx >= 0 && nx < width {
						sum += grid[ny][nx].value
						count++
					}
				}
			}

			avg := sum / float64(count)

			// diffusion + oscillation + noise
			v := grid[y][x].value
			n := (rand.Float64() - 0.5) * grid[y][x].noise

			v += (avg - v) * 0.25
			v += math.Sin(v*6.28) * 0.01
			v += n

			// clamp
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}

			next[y][x].value = v
			next[y][x].noise = grid[y][x].noise
		}
	}

	grid = next
}

func injectShock() {
	cx := rand.Intn(width)
	cy := rand.Intn(height)

	strength := rand.Float64()*0.8 + 0.2
	radius := rand.Intn(6) + 2

	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			ny := cy + y
			nx := cx + x

			if ny >= 0 && ny < height && nx >= 0 && nx < width {
				dist := math.Sqrt(float64(x*x + y*y))
				if dist <= float64(radius) {
					grid[ny][nx].value = math.Mod(grid[ny][nx].value+strength, 1.0)
				}
			}
		}
	}
}

func render() {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {

			v := grid[y][x].value

			charIndex := int(v * float64(len(chars)))
			if charIndex >= len(chars) {
				charIndex = len(chars) - 1
			}

			colorIndex := int(v * float64(len(colors)))
			if colorIndex >= len(colors) {
				colorIndex = len(colors) - 1
			}

			fmt.Printf("%s%s", colors[colorIndex], chars[charIndex])
		}
		fmt.Print("\x1b[0m\n")
	}
}

func clearScreen() {
	fmt.Print("\x1b[2J\x1b[H")
}

