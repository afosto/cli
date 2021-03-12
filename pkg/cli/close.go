package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// our clean up procedure and exiting the program.
func CloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		os.Exit(0)
	}()
}
