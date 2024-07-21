package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

const tempFile = "/tmp/user_idle_state"

// --
// Creates a temp file if the user is idle
// Removes that file if the user is active
// --
func handleActivityState(treshold int) {
	_, err := exec.Command("rm", tempFile).Output()
	if err != nil {
		fmt.Println("Error deleting file", err)
	}

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
	notifyAfterMin := 30      // notify user to break after X min
	pollTimeSeconds := 60     // the time between each poll, and therefor notification
	idleTimeSeconds := 3 * 60 // 5min is considered idle

	// run the idle checker in the background
	go handleActivityState(idleTimeSeconds)

	// the total time the user has been active
	secondsActive := 0
	for {
		_, err := exec.Command("cat", tempFile).Output()
		if err != nil {
			secondsActive += pollTimeSeconds
			fmt.Println("😅 User active for", secondsActive/60, "min")
		} else {
			secondsActive = 0
			fmt.Println("💤 User idle for", idleTimeSeconds/60, "min")
		}

		if secondsActive >= notifyAfterMin*60 {
			err := sendNotification("🚨 Please Pause! (" + strconv.Itoa(secondsActive/60) + " min active)")
			if err != nil {
				fmt.Println("🔴 Error sending notification", err)
			}
		}

		time.Sleep(time.Duration(pollTimeSeconds) * time.Second)
	}
}
