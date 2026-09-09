package main

import (
	"strings"
)

type cmdResult struct {
	action string
	text   string
}

func handleCommand(input string) cmdResult {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return cmdResult{}
	}

	parts := strings.SplitN(input, " ", 2)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/quit", "/exit", "/q":
		return cmdResult{action: "quit"}
	case "/burn":
		return cmdResult{action: "burn", text: "history cleared"}
	case "/clear":
		return cmdResult{action: "clear"}
	default:
		return cmdResult{action: "error", text: "unknown command: " + cmd}
	}
}
