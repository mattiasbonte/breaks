package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

const (
	tempFile = "/tmp/user_idle_state"
)

// --
// Creates a temp file if the user is idle
// Removes that file if the user is active
// --
func handleActivityState(treshold int) {
	cmd := exec.Command(
		"swayidle",
		"-w",
		"timeout",
		strconv.Itoa(treshold),
		"echo 'idle' > "+tempFile,
		"resume",
		"rm "+tempFile,
	)
	cmd.Run()
}

// --
// Sends a notification to the user
// --
func sendNotification(message string) error {
	cmd := exec.Command("notifyme", "-p", "-m", message)
	return cmd.Run()
}

func main() {
	// --
	// SETTINGS change to user preference
	// --
	notifyAfterMinActive := 30 // notify user to break after X min
	notifyAfterMinIdle := 30   // notify user to start working after X min
	pollTimeSeconds := 60      // the time between each poll, and therefor notification
	idleTimeSeconds := 5 * 60  // time before user is considered idle

	// run the idle checker in the background
	go handleActivityState(idleTimeSeconds)

	secondsActive := 0
	secondsIdle := 0
	for {
		_, err := exec.Command("cat", tempFile).Output()
		if err != nil {
			secondsActive += pollTimeSeconds
			secondsIdle = 0
			fmt.Println("😅 User active for", secondsActive/60, "min")
		} else {
			secondsActive = 0
			secondsIdle += pollTimeSeconds
			fmt.Println("💤 User idle for", secondsIdle/60, "min")
		}

		if secondsActive >= notifyAfterMinActive*60 {
			err := sendNotification("🚨 Self Respect! (" + strconv.Itoa(secondsActive/60) + " min active)")
			if err != nil {
				fmt.Println("🔴 Error sending notification", err)
			}
		}
		if secondsIdle >= notifyAfterMinIdle*60 {
			err := sendNotification("🚨 Start working! (" + strconv.Itoa(secondsIdle/60) + " min idle)")
			if err != nil {
				fmt.Println("🔴 Error sending notification", err)
			}
		}

		time.Sleep(time.Duration(pollTimeSeconds) * time.Second)
	}
}
