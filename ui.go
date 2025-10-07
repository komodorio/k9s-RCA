package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func waitForExit() {
	fmt.Println("\n📋 Press Enter to exit...")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func displayLiveRCAResults(results *RCAPollResponse, pollCount int) {
	fmt.Println("🔍 RCA ANALYSIS IN PROGRESS")
	fmt.Println("====================")
	fmt.Printf("📊 Poll Count: %d | Last Update: %s\n", pollCount, time.Now().Format("15:04:05"))
	fmt.Printf("🆔 Session ID: %s\n", results.SessionID)
	fmt.Printf("✅ Status: %s\n", getStatusText(results.IsComplete))
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

func displayFinalRCAResults(results *RCAPollResponse) {
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

func getStatusText(isComplete bool) string {
	if isComplete {
		return "✅ Complete"
	}
	return "⏳ In Progress"
}

func displayError(message string, err error) {
	fmt.Printf("\n❌ %s: %v\n", message, err)
	waitForExit()
}
