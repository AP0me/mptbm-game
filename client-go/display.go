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
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	BrightRed    = "\033[91m"
	BrightGreen  = "\033[92m"
	BrightYellow = "\033[93m"
	BrightBlue   = "\033[94m"
	BrightCyan   = "\033[96m"
	BrightWhite  = "\033[97m"
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

// ---------------------------------------------------------------------------
// TreeState — local types for unmarshaling the STATE packet.
// The client doesn't import shared, so we mirror the JSON shape here.
// ---------------------------------------------------------------------------

type treeState struct {
	Tree struct {
		Branches    map[string][]string `json:"branches"`
		Selections  map[string]int      `json:"selections"`
		BranchOrder []string            `json:"branch_order"`
	} `json:"tree"`
	EventLogs []string `json:"event_logs"`
}

func wrap(text string, codes ...string) string {
	return strings.Join(codes, "") + text + Reset
}

// PrintTree replaces the old PrintState. It unmarshals the flat
// map[string]any back into a typed treeState and renders the ANSI tree.
func PrintTree(data map[string]any) {
	// Clear screen and home cursor
	fmt.Print("\033[H\033[J")

	// Re-marshal to JSON, then unmarshal into typed struct
	raw, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling state:", err)
		return
	}

	var ts treeState
	if err := json.Unmarshal(raw, &ts); err != nil {
		fmt.Println("Error parsing tree state:", err)
		return
	}

	// Render tree
	fmt.Print(renderTree(&ts))
}

func renderTree(ts *treeState) string {
	var lines []string
	lines = append(lines, "")
	bar := strings.Repeat("=", 60)

	lines = append(lines, wrap(bar, BrightCyan, Bold))
	lines = append(lines, wrap("  Story Tree", BrightCyan, Bold))
	lines = append(lines, wrap(bar, BrightCyan, Bold))
	lines = append(lines, "")

	// Root node
	lines = append(lines, wrap("  root", BrightGreen, Bold))
	lines = append(lines, "   │")

	for bIdx, layer1Key := range ts.Tree.BranchOrder {
		kids := ts.Tree.Branches[layer1Key]
		currentIdx := ts.Tree.Selections[layer1Key]
		isLastBranch := bIdx == len(ts.Tree.BranchOrder)-1

		connector := "├──"
		if isLastBranch {
			connector = "└──"
		}
		lines = append(lines,
			"  "+wrap(connector+" ", Cyan)+wrap(layer1Key, BrightYellow, Bold))

		vpipe := "│  "
		if isLastBranch {
			vpipe = "   "
		}
		vpipeStyled := wrap(vpipe, Cyan)

		for i, kid := range kids {
			isLastKid := i == len(kids)-1
			kidConnector := "├──"
			if isLastKid {
				kidConnector = "└──"
			}
			connectorStyled := wrap(kidConnector, Cyan)

			var marker, kidName string
			if i <= currentIdx {
				marker = wrap("[✓]", Dim, Green)
				kidName = wrap(kid, Green)
			} else {
				marker = wrap("[ ]", Dim, Gray)
				kidName = wrap(kid, Dim)
			}

			lines = append(lines,
				"  "+vpipeStyled+connectorStyled+" "+marker+" "+kidName)
		}

		if !isLastBranch {
			lines = append(lines, wrap("  │", Cyan))
		}
	}

	lines = append(lines, "")
	// Legend
	lines = append(lines,
		wrap("  Legend: ", Dim)+
			wrap("[✓]", Dim, Green)+
			wrap(" done   ", Dim)+
			wrap("[ ]", Dim, Gray)+
			wrap(" not done", Dim))
	lines = append(lines, wrap(bar, BrightCyan, Bold))
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// PrintCards — shows available options (unchanged logic, restyled)
// ---------------------------------------------------------------------------

func PrintCards(cards map[string]any) {
	boxWidth := 44
	fmt.Printf("\n%s%s╔%s╗%s\n", Magenta, Bold, strings.Repeat("═", boxWidth), Reset)
	fmt.Printf("%s%s║%s║%s\n", Bold, Magenta, CenterText("~ AVAILABLE OPTIONS ~", boxWidth), Reset)
	fmt.Printf("%s%s╠%s╣%s\n", Magenta, Bold, strings.Repeat("═", boxWidth), Reset)

	for key, val := range cards {
		cardData := val.(map[string]any)
		name := fmt.Sprintf("%v", cardData["name"])

		cleanText := fmt.Sprintf("[%s] %s", key, name)
		padding := boxWidth - 3 - utf8.RuneCountInString(cleanText)
		if padding < 0 {
			padding = 0
		}

		fmt.Printf("%s%s║%s %s%s%s: %s%s%s%s %s%s%s ║%s\n",
			Magenta, Bold, Reset,
			Bold, Yellow, key,
			Bold, Cyan, name, Reset,
			strings.Repeat(" ", padding),
			Magenta, Bold, Reset)
	}

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
			fmt.Printf("%s%s» %s", White, Bold, Reset)
			TypeWrite(m)
			fmt.Print(Reset)
			time.Sleep(1 * time.Second)
		}
	}
}
