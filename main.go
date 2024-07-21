package main

import (
	"fmt"
	"os/exec"
	"time"
)

func sendNotification(message string) error {
	cmd := exec.Command("notifyme", "-p", "-m", message)
	return cmd.Run()
}

func main() {
	for {
		// Create the swayidle command
		cmd := exec.Command("swayidle", "-w", "timeout", "60", "echo 'idle'")

		// Use a channel to capture the 'idle' output or termination.
		done := make(chan error, 1)
		go func() {
			err := cmd.Run()
			done <- err // Send nil if no error occurs, else send error
		}()

		select {
		case err := <-done:
			if err != nil {
				fmt.Println("Error running swayidle:", err)
				return
			}
			// User was idle, swayidle completed, wait for activity
			for {
				// Wait until the user gets active
				time.Sleep(1 * time.Second)
				cmdCheckActive := exec.Command("swaymsg", "-t", "get_tree")
				err = cmdCheckActive.Run()
				if err == nil {
					// User is active
					time.Sleep(1 * time.Minute)
					sendNotification("You have been active for more than 1 minute!")
					break
				}
			}
		}
	}
}
