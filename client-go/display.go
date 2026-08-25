package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
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

func runeLength(text string) int {
	return utf8.RuneCountInString(text)
}

func CenterText(text string, width int) string {
	length := runeLength(text)
	if length >= width {
		return text
	}

	left := (width - length) / 2
	right := width - length - left

	return strings.Repeat(" ", left) +
		text +
		strings.Repeat(" ", right)
}

func terminalWidth() int {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
			return width
		}
	}

	return 100
}

// boxWidth chooses an inner box width that fits inside the terminal.
func boxWidth(preferred int) int {
	width := terminalWidth()

	if width <= 4 {
		return 1
	}

	maximum := width - 2
	if preferred > maximum {
		return maximum
	}

	return preferred
}

func boxIndent(innerWidth int) string {
	padding := (terminalWidth() - (innerWidth + 2)) / 2
	if padding < 0 {
		padding = 0
	}

	return strings.Repeat(" ", padding)
}

func truncateRunes(text string, maximum int) string {
	if maximum <= 0 {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= maximum {
		return text
	}

	if maximum <= 3 {
		return string(runes[:maximum])
	}

	return string(runes[:maximum-3]) + "..."
}

// WrapText wraps text by rune count and splits very long words when needed.
func WrapText(text string, width int) []string {
	if width <= 0 {
		return []string{""}
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}

	remaining := []rune(text)
	lines := make([]string, 0)

	for len(remaining) > width {
		cut := width

		// Prefer breaking at the final whitespace within the line.
		for i := width; i > 0; i-- {
			if unicode.IsSpace(remaining[i-1]) {
				cut = i - 1
				break
			}
		}

		// The first word is longer than the available width.
		if cut == 0 {
			cut = width
		}

		line := strings.TrimSpace(string(remaining[:cut]))
		if line != "" {
			lines = append(lines, line)
		}

		remaining = remaining[cut:]
		for len(remaining) > 0 && unicode.IsSpace(remaining[0]) {
			remaining = remaining[1:]
		}
	}

	if len(remaining) > 0 {
		lines = append(lines, strings.TrimSpace(string(remaining)))
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}

func printBoxBorder(indent, color, left, fill, right string, width int) {
	fmt.Printf(
		"%s%s%s%s%s%s%s%s\n",
		indent,
		color,
		Bold,
		left,
		strings.Repeat(fill, width),
		right,
		Reset,
		"",
	)
}

// printBoxLine prints ANSI-colored content while using visibleWidth for
// padding calculations.
func printBoxLine(
	indent string,
	color string,
	content string,
	visibleWidth int,
	width int,
) {
	padding := width - visibleWidth
	if padding < 0 {
		padding = 0
	}

	fmt.Printf(
		"%s%s%s║%s%s%s%s%s║%s\n",
		indent,
		color,
		Bold,
		Reset,
		content,
		strings.Repeat(" ", padding),
		color,
		Bold,
		Reset,
	)
}

func printBoxTitle(indent, color, title string, width int) {
	title = truncateRunes(title, width)
	left := (width - runeLength(title)) / 2
	right := width - runeLength(title) - left

	content := strings.Repeat(" ", left) +
		Bold + White + title + Reset +
		strings.Repeat(" ", right)

	printBoxLine(indent, color, content, width, width)
}

func GetStringList(state map[string]any, key string) []string {
	switch value := state[key].(type) {
	case []any:
		list := make([]string, len(value))
		for i, item := range value {
			list[i] = fmt.Sprintf("%v", item)
		}
		return list

	case []string:
		return value

	default:
		return []string{}
	}
}

func stateValueString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}

	return string(data)
}

func PrintState(state map[string]any) {
	displayOrder := GetStringList(state, "display_order")

	// Clear screen and move the cursor home.
	fmt.Print("\033[H\033[J")

	width := boxWidth(64)
	indent := boxIndent(width)
	contentWidth := width - 2

	if contentWidth < 1 {
		contentWidth = 1
	}

	printBoxBorder(indent, Cyan, "╔", "═", "╗", width)
	printBoxTitle(indent, Cyan, "~ GAME STATE ~", width)
	printBoxBorder(indent, Cyan, "╠", "═", "╣", width)

	if len(displayOrder) == 0 {
		message := "No state values"
		message = truncateRunes(message, contentWidth)

		content := " " + Gray + Dim + message + Reset
		printBoxLine(
			indent,
			Cyan,
			content,
			1+runeLength(message),
			width,
		)
	}

	for _, key := range displayOrder {
		valueText := stateValueString(state[key])

		// Reserve most of the line for the value.
		maximumKeyWidth := contentWidth / 3
		if maximumKeyWidth < 1 {
			maximumKeyWidth = 1
		}

		displayKey := truncateRunes(key, maximumKeyWidth)
		label := displayKey + ": "
		labelWidth := runeLength(label)

		valueWidth := contentWidth - labelWidth
		if valueWidth < 1 {
			valueWidth = 1
		}

		lines := WrapText(valueText, valueWidth)

		for index, line := range lines {
			var content string
			var visibleWidth int

			if index == 0 {
				content = " " +
					Bold + Yellow + label + Reset +
					Green + line + Reset

				visibleWidth = 1 + labelWidth + runeLength(line)
			} else {
				content = " " +
					strings.Repeat(" ", labelWidth) +
					Green + line + Reset

				visibleWidth = 1 + labelWidth + runeLength(line)
			}

			printBoxLine(
				indent,
				Cyan,
				content,
				visibleWidth,
				width,
			)
		}
	}

	printBoxBorder(indent, Cyan, "╚", "═", "╝", width)
}

func PrintCards(cards map[string]any) {
	width := boxWidth(54)
	indent := boxIndent(width)
	contentWidth := width - 2

	if contentWidth < 1 {
		contentWidth = 1
	}

	fmt.Println()

	printBoxBorder(indent, Magenta, "╔", "═", "╗", width)
	printBoxTitle(indent, Magenta, "~ PLAYABLE CARDS ~", width)
	printBoxBorder(indent, Magenta, "╠", "═", "╣", width)

	// Map iteration order is random, so sort choice keys for stable output.
	keys := make([]string, 0, len(cards))
	for key := range cards {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		message := "No playable cards"
		message = truncateRunes(message, contentWidth)

		content := " " + Gray + Dim + message + Reset
		printBoxLine(
			indent,
			Magenta,
			content,
			1+runeLength(message),
			width,
		)
	}

	for _, key := range keys {
		name := cardName(cards[key])

		maximumKeyWidth := contentWidth / 3
		if maximumKeyWidth < 1 {
			maximumKeyWidth = 1
		}

		displayKey := truncateRunes(key, maximumKeyWidth)
		label := displayKey + ": "
		labelWidth := runeLength(label)

		nameWidth := contentWidth - labelWidth
		if nameWidth < 1 {
			nameWidth = 1
		}

		// Long card names wrap with a hanging indent.
		nameLines := WrapText(name, nameWidth)

		for index, line := range nameLines {
			var content string
			var visibleWidth int

			if index == 0 {
				content = " " +
					Bold + Yellow + label + Reset +
					Bold + Cyan + line + Reset

				visibleWidth = 1 + labelWidth + runeLength(line)
			} else {
				content = " " +
					strings.Repeat(" ", labelWidth) +
					Cyan + line + Reset

				visibleWidth = 1 + labelWidth + runeLength(line)
			}

			printBoxLine(
				indent,
				Magenta,
				content,
				visibleWidth,
				width,
			)
		}
	}

	printBoxBorder(indent, Magenta, "╚", "═", "╝", width)
	fmt.Println()
}

func cardName(value any) string {
	cardData, ok := value.(map[string]any)
	if !ok {
		return fmt.Sprintf("%v", value)
	}

	name, ok := cardData["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return "Unnamed card"
	}

	return name
}

func TypeWrite(text string) {
	for _, char := range text {
		fmt.Print(string(char))
		time.Sleep(30 * time.Millisecond)
	}

	fmt.Println()
}

func PrintCardResponse(messages []any) {
	for _, message := range messages {
		text, ok := message.(string)
		if !ok {
			continue
		}

		fmt.Printf("%s%s» ", White, Bold)
		TypeWrite(text)
		fmt.Print(Reset)

		time.Sleep(time.Second)
		fmt.Print("Press 'Enter' to continue...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
}
