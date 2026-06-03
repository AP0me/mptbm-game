package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"
)

func PrintError(message string) {
	fmt.Printf("\n%s%s[ERROR]: %s%s\n\n", Bold, Red, message, Reset)
}

func CenterText(text string, width int) string {
	length := utf8.RuneCountInString(text)
	if length >= width {
		return text
	}
	paddingLeft := (width - length) / 2
	paddingRight := width - length - paddingLeft
	return strings.Repeat(" ", paddingLeft) + text + strings.Repeat(" ", paddingRight)
}

func PrintState(state map[string]any) {
	display_order := state["display_order"].([]string)
	// Clear screen and home cursor
	fmt.Print("\033[H\033[J")

	boxWidth := 60
	fmt.Printf("%s%s╔%s╗%s\n", Cyan, Bold, strings.Repeat("═", boxWidth), Reset)
	fmt.Printf("%s%s║%s║%s\n", Bold, Cyan, CenterText("~ GAME STATE ~", boxWidth), Reset)
	fmt.Printf("%s%s╠%s╣%s\n", Cyan, Bold, strings.Repeat("═", boxWidth), Reset)

	for _, k := range display_order {
		val := state[k]
		valBytes, _ := json.Marshal(val)
		valStr := string(valBytes)

		if utf8.RuneCountInString(valStr) > 45 {
			valStr = valStr[:42] + "..."
		}

		keyPart := fmt.Sprintf("%s%s%s%s: ", Bold, Yellow, k, Reset)
		valPart := fmt.Sprintf("%s%s%s", Green, valStr, Reset)

		// Calculate padding (ignoring ANSI codes)
		cleanLen := utf8.RuneCountInString(k) + 2 + utf8.RuneCountInString(valStr)
		padding := boxWidth - 2 - cleanLen
		if padding < 0 {
			padding = 0
		}

		fmt.Printf("%s%s║%s %s%s%s %s%s║%s\n", Cyan, Bold, Reset, keyPart, valPart, strings.Repeat(" ", padding), Cyan, Bold, Reset)
	}

	fmt.Printf("%s%s╚%s╝%s\n", Cyan, Bold, strings.Repeat("═", boxWidth), Reset)
}

func PrintCards(cards map[string]any) {
	boxWidth := 44
	// Top Border: 4 placeholders, 4 variables
	fmt.Printf("\n%s%s╔%s╗%s\n", Magenta, Bold, strings.Repeat("═", boxWidth), Reset)
	fmt.Printf("%s%s║%s║%s\n", Bold, Magenta, CenterText("~ PLAYABLE CARDS ~", boxWidth), Reset)
	fmt.Printf("%s%s╠%s╣%s\n", Magenta, Bold, strings.Repeat("═", boxWidth), Reset)

	for key, val := range cards {
		cardData := val.(map[string]any)
		name := cardData["name"].(string)

		cleanText := fmt.Sprintf("%s: %s", key, name)
		padding := boxWidth - 3 - utf8.RuneCountInString(cleanText)
		if padding < 0 {
			padding = 0
		}

		// Line Content: 14 placeholders, 14 variables
		// 1(%s) 2(%s) ║ 3(%s) 4(%s) 5(%s) 6(%s) : 7(%s) 8(%s) 9(%s) 10(%s) 11(%s) 12(%s) 13(%s) ║ 14(%s)
		fmt.Printf("%s%s║%s %s%s%s: %s%s%s%s %s%s%s ║%s\n",
			Magenta, Bold, Reset, // 1, 2, 3
			Bold, Yellow, key, // 4, 5, 6
			Bold, Cyan, name, Reset, // 7, 8, 9, 10
			strings.Repeat(" ", padding), Magenta, Bold, // 11, 12, 13
			Reset) // 14
	}

	// Bottom Border: 4 placeholders, 4 variables
	fmt.Printf("%s%s╚%s╝%s\n\n", Magenta, Bold, strings.Repeat("═", boxWidth), Reset)
}

func TypeWrite(text string) {
	for _, char := range text {
		fmt.Print(string(char))
		time.Sleep(30 * time.Millisecond)
	}
	fmt.Println()
}

func PrintCardResponse(messages []any) {
	for _, msg := range messages {
		if m, ok := msg.(string); ok {
			fmt.Printf("%s%s» ", White, Bold)
			TypeWrite(m)
			fmt.Print(Reset)
			time.Sleep(1 * time.Second)
		}
	}
}
