package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

type ConsoleTUI struct{}

func NewConsoleTUI() *ConsoleTUI {
	return &ConsoleTUI{}
}

func (c *ConsoleTUI) ClearScreen() {
	fmt.Print("\033[H\033[2J")
}

func (c *ConsoleTUI) DisplayLiveRCAResults(results *RCAPollResponse, pollCount int) {
	fmt.Println("🔍 RCA ANALYSIS IN PROGRESS")
	fmt.Println("====================")
	fmt.Printf("📊 Poll Count: %d | Last Update: %s\n", pollCount, time.Now().Format("15:04:05"))
	fmt.Printf("🆔 Session ID: %s\n", results.SessionID)
	fmt.Printf("✅ Status: %s\n", c.getStatusText(results.IsComplete))
	fmt.Println()

	if results.ProblemShort != "" {
		fmt.Printf("📋 Problem: %s\n", results.ProblemShort)
	}

	if results.Recommendation != "" {
		fmt.Printf("💡 Recommendation: %s\n", results.Recommendation)
	}

	fmt.Println()
	fmt.Println("📝 What Happened:")
	if len(results.WhatHappened) > 0 {
		for i, event := range results.WhatHappened {
			fmt.Printf("  %d. %s\n", i+1, event)
		}
	} else {
		fmt.Println("  ⏳ Waiting for data...")
	}

	fmt.Println()
	fmt.Println("🔍 Evidence:")
	if len(results.EvidenceQueries) > 0 {
		for i, query := range results.EvidenceQueries {
			fmt.Printf("  %d. %s\n", i+1, query)
		}
	} else {
		fmt.Println("  ⏳ Waiting for data...")
	}

	fmt.Println()
	fmt.Println("📊 Operations:")
	if len(results.Operations) > 0 {
		for i, operation := range results.Operations {
			fmt.Printf("  %d. %s\n", i+1, operation)
		}
	} else {
		fmt.Println("  ⏳ Waiting for data...")
	}

	fmt.Println()
	fmt.Println("====================")
	fmt.Println("Press Ctrl+C to stop monitoring")
}

func (c *ConsoleTUI) DisplayFinalRCAResults(results *RCAPollResponse) {
	fmt.Println("✅ RCA ANALYSIS COMPLETED!")
	fmt.Println("==========================")
	fmt.Printf("🆔 Session ID: %s\n", results.SessionID)
	fmt.Printf("⏰ Completed at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println()

	if results.ProblemShort != "" {
		fmt.Printf("📋 Problem: %s\n", results.ProblemShort)
	}

	if results.Recommendation != "" {
		fmt.Printf("💡 Recommendation: %s\n", results.Recommendation)
	}

	fmt.Println()
	fmt.Println("📝 What Happened:")
	if len(results.WhatHappened) > 0 {
		for i, event := range results.WhatHappened {
			fmt.Printf("  %d. %s\n", i+1, event)
		}
	} else {
		fmt.Println("  • No what happened data available")
	}

	fmt.Println()
	fmt.Println("🔍 Evidence:")
	if len(results.EvidenceQueries) > 0 {
		for i, query := range results.EvidenceQueries {
			fmt.Printf("  %d. %s\n", i+1, query)
		}
	} else {
		fmt.Println("  • No evidence queries found")
	}

	fmt.Println()
	fmt.Println("📊 Operations Performed:")
	if len(results.Operations) > 0 {
		for i, operation := range results.Operations {
			fmt.Printf("  %d. %s\n", i+1, operation)
		}
	} else {
		fmt.Println("  • No operations data available")
	}

	fmt.Println()
	fmt.Println("==========================")
}

func (c *ConsoleTUI) DisplayError(message string, err error) {
	fmt.Printf("\n❌ %s: %v\n", message, err)
	c.WaitForExit()
}

func (c *ConsoleTUI) DisplayMessage(message string) {
	fmt.Println(message)
}

func (c *ConsoleTUI) DisplayProgressIndicator(message string) {
	fmt.Printf("\r%s", message)
}

func (c *ConsoleTUI) WaitForExit() {
	fmt.Println("\n📋 Press Enter to exit...")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
}

func (c *ConsoleTUI) getStatusText(isComplete bool) string {
	if isComplete {
		return "✅ Complete"
	}
	return "⏳ In Progress"
}
