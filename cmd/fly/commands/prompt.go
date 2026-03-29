package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// IsInteractive returns true if stdin is a terminal.
func IsInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// IsColorTerminal returns true if stdout supports ANSI color codes.
func IsColorTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Prompt asks the user a question and returns their answer.
func Prompt(question, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", question, defaultVal)
	} else {
		fmt.Printf("%s: ", question)
	}
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return defaultVal
	}
	return answer
}

// PromptSelect asks the user to choose from a list of options.
func PromptSelect(question string, options []string, defaultVal string) string {
	fmt.Printf("%s\n", question)
	for i, opt := range options {
		marker := " "
		if opt == defaultVal {
			marker = ">"
		}
		fmt.Printf("  %s %d) %s\n", marker, i+1, opt)
	}
	if defaultVal != "" {
		fmt.Printf("Choice [%s]: ", defaultVal)
	} else {
		fmt.Printf("Choice: ")
	}
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return defaultVal
	}
	for i, opt := range options {
		if answer == fmt.Sprintf("%d", i+1) || strings.EqualFold(answer, opt) {
			return opt
		}
	}
	return answer
}

// PromptConfirm asks a yes/no question.
func PromptConfirm(question string, defaultYes bool) bool {
	hint := "y/N"
	if defaultYes {
		hint = "Y/n"
	}
	fmt.Printf("%s [%s]: ", question, hint)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "" {
		return defaultYes
	}
	return answer == "y" || answer == "yes"
}
