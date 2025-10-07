package main

type TUI interface {
	ClearScreen()
	DisplayLiveRCAResults(results *RCAPollResponse, pollCount int)
	DisplayFinalRCAResults(results *RCAPollResponse)
	DisplayError(message string, err error)
	DisplayMessage(message string)
	DisplayProgressIndicator(message string)
	WaitForExit()
}
