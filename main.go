package main

import (
	"bufio"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	width          = 12
	height         = 12
	preSimTicks    = 75
	tickDuration   = 2 * time.Second
	maxPopulation  = 255
	minStability   = 0.0
	maxStability   = 1.0
	pressureClamp  = 1.0
	influenceRegen = 1
)

type Alignment string

const (
	Neutral       Alignment = "Neutral"
	Player        Alignment = "Player"
	Rival         Alignment = "Rival"
	Institutional Alignment = "Institutional"
)

type Cell struct {
	pressure         float64
	stability        float64
	population       int
	institutional    float64
	logisticsBoost   float64
	investmentTicks  int
	suppressionTicks int
	backlashTicks    int
	alignment        Alignment
}

type Card struct {
	name     string
	cost     int
	cooldown int
	current  int
	play     func(x, y int) string
	valid    func(x, y int) bool
}

type Command struct {
	card string
	x    int
	y    int
}

var (
	grid      [height][width]Cell
	nextGrid  [height][width]Cell
	tickCount int
	influence int
	eventLog  []string
	cards     []Card
)

var (
	alignmentColors = map[Alignment]string{
		Neutral:       "\x1b[37m",
		Player:        "\x1b[32m",
		Rival:         "\x1b[31m",
		Institutional: "\x1b[34m",
	}
	intensityRunes = []string{" ", "·", "•", "◦", "◎", "◉", "●"}
)

func main() {
	rand.Seed(time.Now().UnixNano())
	influence = 10
	initGrid()
	initCards()
	seedWorld()

	commands := make(chan Command, 8)
	go readCommands(commands)

	for {
		start := time.Now()
		clearScreen()
		processCommands(commands)
		stepSimulation()
		render()
		sleepRemaining(start)
	}
}

func initGrid() {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			grid[y][x] = Cell{
				pressure:      rand.Float64()*0.4 - 0.2,
				stability:     rand.Float64()*0.4 + 0.4,
				population:    rand.Intn(120) + 40,
				institutional: rand.Float64() * 0.3,
				alignment:     Neutral,
			}
		}
	}
}

func seedWorld() {
	for i := 0; i < preSimTicks; i++ {
		stepSimulation()
	}
}

func initCards() {
	cards = []Card{
		{
			name:     "Outreach",
			cost:     1,
			cooldown: 2,
			play:     playOutreach,
			valid:    func(x, y int) bool { return inBounds(x, y) },
		},
		{
			name:     "Investment",
			cost:     2,
			cooldown: 3,
			play:     playInvestment,
			valid:    func(x, y int) bool { return inBounds(x, y) && grid[y][x].population < 200 },
		},
		{
			name:     "Enforcement",
			cost:     2,
			cooldown: 3,
			play:     playEnforcement,
			valid:    func(x, y int) bool { return inBounds(x, y) && grid[y][x].pressure < 0.1 },
		},
		{
			name:     "Logistics",
			cost:     1,
			cooldown: 4,
			play:     playLogistics,
			valid:    func(x, y int) bool { return inBounds(x, y) && grid[y][x].logisticsBoost < 0.5 },
		},
		{
			name:     "Suppression",
			cost:     3,
			cooldown: 5,
			play:     playSuppression,
			valid:    func(x, y int) bool { return inBounds(x, y) && grid[y][x].institutional > 0.1 },
		},
	}
}

func readCommands(commands chan<- Command) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) < 3 {
			continue
		}
		card := fields[0]
		x, errX := strconv.Atoi(fields[1])
		y, errY := strconv.Atoi(fields[2])
		if errX != nil || errY != nil {
			continue
		}
		commands <- Command{card: card, x: x, y: y}
	}
}

func processCommands(commands <-chan Command) {
	for {
		select {
		case cmd := <-commands:
			applyCommand(cmd)
		default:
			return
		}
	}
}

func applyCommand(cmd Command) {
	cardIndex := findCard(cmd.card)
	if cardIndex == -1 {
		appendEvent("Unknown card: "+cmd.card, "")
		return
	}
	card := &cards[cardIndex]
	if card.current > 0 {
		appendEvent(card.name+" is cooling down.", "")
		return
	}
	if influence < card.cost {
		appendEvent("Not enough influence for "+card.name+".", "")
		return
	}
	if !card.valid(cmd.x, cmd.y) {
		appendEvent("Invalid placement for "+card.name+".", "")
		return
	}
	message := card.play(cmd.x, cmd.y)
	influence -= card.cost
	card.current = card.cooldown
	appendEvent(message, fmt.Sprintf("(%d,%d)", cmd.x, cmd.y))
}

func findCard(input string) int {
	for i, card := range cards {
		if strings.EqualFold(card.name, input) {
			return i
		}
		if strconv.Itoa(i+1) == input {
			return i
		}
	}
	return -1
}

func stepSimulation() {
	tickCount++
	for i := range cards {
		if cards[i].current > 0 {
			cards[i].current--
		}
	}
	influence += influenceRegen
	if influence > 20 {
		influence = 20
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := grid[y][x]
			neighborPressure := averageNeighborPressure(x, y)
			spreadRate := 0.15 + cell.logisticsBoost
			pressure := cell.pressure + (neighborPressure-cell.pressure)*spreadRate
			pressure *= 0.98

			stability := cell.stability
			stability += (0.5 - stability) * 0.02
			stability -= math.Abs(pressure) * 0.05
			stability = clamp(stability, minStability, maxStability)

			population := cell.population
			if stability > 0.6 {
				population += 1
			}
			if stability < 0.3 {
				population -= 1
			}
			if population < 0 {
				population = 0
			}
			if population > maxPopulation {
				population = maxPopulation
			}

			institutional := cell.institutional
			institutional += math.Abs(pressure) * 0.04
			institutional += cell.logisticsBoost * 0.03
			institutional -= 0.01

			if cell.investmentTicks > 0 {
				institutional += 0.02
				cell.investmentTicks--
			}

			if cell.suppressionTicks > 0 {
				institutional -= 0.06
				cell.suppressionTicks--
			}
			if cell.suppressionTicks == 0 && cell.backlashTicks > 0 {
				institutional += 0.05
				stability = clamp(stability-0.02, minStability, maxStability)
				cell.backlashTicks--
			}

			institutional = clamp(institutional, 0, 1)
			pressure = clamp(pressure, -pressureClamp, pressureClamp)

			nextGrid[y][x] = Cell{
				pressure:         pressure,
				stability:        stability,
				population:       population,
				institutional:    institutional,
				logisticsBoost:   math.Max(0, cell.logisticsBoost-0.01),
				investmentTicks:  cell.investmentTicks,
				suppressionTicks: cell.suppressionTicks,
				backlashTicks:    cell.backlashTicks,
				alignment:        deriveAlignment(pressure, stability, institutional),
			}
		}
	}
	grid = nextGrid
}

func averageNeighborPressure(x, y int) float64 {
	sum := 0.0
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx := x + dx
			ny := y + dy
			if inBounds(nx, ny) {
				sum += grid[ny][nx].pressure
				count++
			}
		}
	}
	return sum / float64(count)
}

func deriveAlignment(pressure, stability, institutional float64) Alignment {
	if institutional > 0.6 && math.Abs(pressure) < 0.2 || stability < 0.2 {
		return Institutional
	}
	if pressure > 0.2 {
		return Player
	}
	if pressure < -0.2 {
		return Rival
	}
	return Neutral
}

func render() {
	fmt.Printf("TRAPX v1 | Tick %d | Influence %d/20\n", tickCount, influence)
	fmt.Println("Commands: <card> <x> <y> (card by name or 1-5). Grid coords: 0-11.")
	fmt.Println("Hand:")
	for i, card := range cards {
		status := "ready"
		if card.current > 0 {
			status = fmt.Sprintf("cooldown %d", card.current)
		}
		fmt.Printf("%d) %s (cost %d, %s)  ", i+1, card.name, card.cost, status)
	}
	fmt.Println("\n")

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := grid[y][x]
			intensity := pressureIntensity(cell)
			color := alignmentColors[cell.alignment]
			fmt.Printf("%s%s", color, intensity)
		}
		fmt.Print("\x1b[0m")
		fmt.Printf(" | %d\n", y)
	}

	fmt.Println("\nLegend: Player (green), Rival (red), Institutional (blue), Neutral (white).")
	fmt.Println("Intensity = pressure or institutional force. Stability/population drive drift every tick.")
	printEvents()
}

func pressureIntensity(cell Cell) string {
	value := math.Abs(cell.pressure)
	if cell.alignment == Institutional {
		value = cell.institutional
	}
	index := int(value * float64(len(intensityRunes)))
	if index >= len(intensityRunes) {
		index = len(intensityRunes) - 1
	}
	return intensityRunes[index]
}

func printEvents() {
	if len(eventLog) == 0 {
		fmt.Println("\nRecent actions: none")
		return
	}
	fmt.Println("\nRecent actions:")
	for _, entry := range eventLog {
		fmt.Println("-", entry)
	}
}

func appendEvent(message, location string) {
	if location != "" {
		message = fmt.Sprintf("%s %s", message, location)
	}
	eventLog = append(eventLog, message)
	if len(eventLog) > 4 {
		eventLog = eventLog[len(eventLog)-4:]
	}
}

func playOutreach(x, y int) string {
	cell := grid[y][x]
	cell.stability = clamp(cell.stability+0.1, minStability, maxStability)
	cell.pressure = clamp(cell.pressure+0.05, -pressureClamp, pressureClamp)
	grid[y][x] = cell
	return "Outreach increased stability and pressure"
}

func playInvestment(x, y int) string {
	cell := grid[y][x]
	cell.population = min(cell.population+12, maxPopulation)
	cell.stability = clamp(cell.stability+0.05, minStability, maxStability)
	cell.investmentTicks += 5
	grid[y][x] = cell
	return "Investment boosted population and stability"
}

func playEnforcement(x, y int) string {
	cell := grid[y][x]
	if cell.pressure < 0 {
		cell.pressure = clamp(cell.pressure+0.25, -pressureClamp, pressureClamp)
	}
	cell.stability = clamp(cell.stability-0.1, minStability, maxStability)
	cell.institutional = clamp(cell.institutional+0.05, 0, 1)
	grid[y][x] = cell
	return "Enforcement reduced rival pressure"
}

func playLogistics(x, y int) string {
	cell := grid[y][x]
	cell.logisticsBoost = clamp(cell.logisticsBoost+0.2, 0, 0.5)
	cell.institutional = clamp(cell.institutional+0.03, 0, 1)
	grid[y][x] = cell
	return "Logistics increased pressure spread"
}

func playSuppression(x, y int) string {
	cell := grid[y][x]
	cell.institutional = clamp(cell.institutional-0.2, 0, 1)
	cell.suppressionTicks = 3
	cell.backlashTicks += 3
	grid[y][x] = cell
	return "Suppression reduced institutional pressure"
}

func inBounds(x, y int) bool {
	return x >= 0 && x < width && y >= 0 && y < height
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clearScreen() {
	fmt.Print("\x1b[2J\x1b[H")
}

func sleepRemaining(start time.Time) {
	elapsed := time.Since(start)
	if elapsed < tickDuration {
		time.Sleep(tickDuration - elapsed)
	}
}
