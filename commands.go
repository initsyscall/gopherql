package main

import (
	"strings"
)

type cmdResult struct {
	action string
	text   string
	target string
}

var allCommands = []string{
	"/quit",
	"/burn history",
	"/burn db",
	"/clear",
}

func filterCommands(input string) []string {
	input = strings.ToLower(strings.TrimSpace(input))
	var matches []string
	for _, c := range allCommands {
		if input == "" || strings.Contains(strings.ToLower(c), input) {
			matches = append(matches, c)
		}
	}
	return matches
}

func handleCommand(input string) cmdResult {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return cmdResult{}
	}

	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/quit", "/exit", "/q":
		return cmdResult{action: "quit"}
	case "/burn":
		target := ""
		if len(parts) > 1 {
			target = strings.ToLower(parts[1])
		}
		switch target {
		case "history":
			return cmdResult{action: "burn", target: "history", text: "history burned"}
		case "db":
			return cmdResult{action: "burn", target: "db", text: "database burned"}
		case "":
			return cmdResult{action: "error", text: "usage: /burn <history|db>"}
		default:
			return cmdResult{action: "error", text: "unknown burn target: " + target}
		}
	case "/clear":
		return cmdResult{action: "clear"}
	default:
		return cmdResult{action: "error", text: "unknown command: " + cmd}
	}
}
