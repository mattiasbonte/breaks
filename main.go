package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	userIdleStateFile    = "/tmp/user_idle_state"
	userPreIdleStateFile = "/tmp/user_pre_idle_state"
	secondsUntilIdle     = 7 * 30 // 3min30s
)

type DisplayServer int

const (
	Wayland DisplayServer = iota
	X11
	Unknown
)

// detectDisplayServer determines if we're running on X11 or Wayland
func detectDisplayServer() DisplayServer {
	// Check for Wayland
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return Wayland
	}
	
	// Check for X11
	if os.Getenv("DISPLAY") != "" {
		return X11
	}
	
	return Unknown
}

// MAIN SCRIPT
func main() {
	// --
	// SETTINGS SET AS PREFERRED
	// --
	notifyAfterMinActive := 30 // notify user to break after X min
	notifyAfterMinIdle := 60   // notify user to start working after X min
	pollTimeSeconds := 60      // the time between each poll, and therefor notification

	go trackPreIdleState()
	go trackIdleState()

	trackActivity(notifyAfterMinActive, notifyAfterMinIdle, pollTimeSeconds)

}

// Track the user's idle state
// A file is created if the idle treshold is reached
// The file acts as a flag to indicate that the user is indeed idle
func trackIdleState() {
	displayServer := detectDisplayServer()
	
	switch displayServer {
	case Wayland:
		cmd := exec.Command(
			"swayidle",
			"-w",
			"timeout",
			strconv.Itoa(secondsUntilIdle),
			"echo 'idle' > "+userIdleStateFile,
			"resume",
			"rm "+userIdleStateFile,
		)
		cmd.Run()
	case X11:
		trackIdleStateX11(secondsUntilIdle)
	default:
		fmt.Println("🔴 Unknown display server, cannot track idle state")
	}
}

// trackIdleStateX11 tracks idle state on X11 using xprintidle
func trackIdleStateX11(idleThreshold int) {
	for {
		cmd := exec.Command("xprintidle")
		output, err := cmd.Output()
		if err != nil {
			fmt.Println("🔴 Error running xprintidle:", err)
			time.Sleep(1 * time.Second)
			continue
		}
		
		idleMs, err := strconv.Atoi(strings.TrimSpace(string(output)))
		if err != nil {
			fmt.Println("🔴 Error parsing xprintidle output:", err)
			time.Sleep(1 * time.Second)
			continue
		}
		
		idleSeconds := idleMs / 1000
		
		if idleSeconds >= idleThreshold {
			// Create idle state file
			exec.Command("sh", "-c", "echo 'idle' > "+userIdleStateFile).Run()
		} else {
			// Remove idle state file
			exec.Command("rm", userIdleStateFile).Run()
		}
		
		time.Sleep(1 * time.Second)
	}
}

// trackPreIdleStateX11 tracks pre-idle state on X11 using xprintidle
func trackPreIdleStateX11(idleThreshold int) {
	for {
		cmd := exec.Command("xprintidle")
		output, err := cmd.Output()
		if err != nil {
			fmt.Println("🔴 Error running xprintidle:", err)
			time.Sleep(1 * time.Second)
			continue
		}
		
		idleMs, err := strconv.Atoi(strings.TrimSpace(string(output)))
		if err != nil {
			fmt.Println("🔴 Error parsing xprintidle output:", err)
			time.Sleep(1 * time.Second)
			continue
		}
		
		idleSeconds := idleMs / 1000
		
		if idleSeconds >= idleThreshold {
			// Create pre-idle state file
			exec.Command("sh", "-c", "echo 'idle' > "+userPreIdleStateFile).Run()
		} else {
			// Remove pre-idle state file
			exec.Command("rm", userPreIdleStateFile).Run()
		}
		
		time.Sleep(1 * time.Second)
	}
}

// Shorter version of the trackIdleState function
// Which holds a shorter leash on the idle state but is more sensitive
func trackPreIdleState() {
	secondsUntilPreIdle := 30
	displayServer := detectDisplayServer()
	
	switch displayServer {
	case Wayland:
		cmd := exec.Command(
			"swayidle",
			"-w",
			"timeout",
			strconv.Itoa(secondsUntilPreIdle),
			"echo 'idle' > "+userPreIdleStateFile,
			"resume",
			"rm "+userPreIdleStateFile,
		)
		cmd.Run()
	case X11:
		trackPreIdleStateX11(secondsUntilPreIdle)
	default:
		fmt.Println("🔴 Unknown display server, cannot track pre-idle state")
	}
}

func notifyUser(message string) error {
	cmd := exec.Command("notifyme", "-p", "-m", message)
	return cmd.Run()
}

func trackActivity(
	notifyAfterMinActive int,
	notifyAfterMinIdle int,
	pollTimeSeconds int,
) {
	type Active struct{ seconds int }
	type Idle struct{ seconds int }
	type PreIdle struct{ seconds int }
	type Notified struct {
		active bool
		idle   bool
	}
	type State struct {
		active   Active
		idle     Idle
		preIdle  PreIdle
		notified Notified
	}
	s := State{
		active:   Active{seconds: 0},
		idle:     Idle{seconds: 0},
		preIdle:  PreIdle{seconds: 0},
		notified: Notified{active: false, idle: false},
	}

	for {
		// check idle state
		_, err := exec.Command("cat", userIdleStateFile).Output()
		if err != nil {
			s.active.seconds += pollTimeSeconds
			s.idle.seconds = secondsUntilIdle
			s.notified.idle = false
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "😅 Active for", s.active.seconds/60, "min")
		} else {
			s.active.seconds = 0
			s.idle.seconds += pollTimeSeconds
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "💤 Idle for", s.idle.seconds/60, "min")
		}

		// check pre idle state
		_, err = exec.Command("cat", userPreIdleStateFile).Output()
		if err == nil {
			s.preIdle.seconds += pollTimeSeconds
		} else {
			s.preIdle.seconds = 0
		}

		// notify user if active past given treshold
		if s.active.seconds >= notifyAfterMinActive*60 {
			if s.preIdle.seconds == 0 {
				err := notifyUser("🚨 Self Respect! (" + strconv.Itoa(s.active.seconds/60) + " min active)")
				if err != nil {
					fmt.Println("🔴 Error sending notification", err)
				}
			}
		}

		// notify user if idle past given treshold
		if s.idle.seconds >= notifyAfterMinIdle*60 {
			if !s.notified.idle {
				err := notifyUser("🦋 Gentle reminder to start working again? (" + strconv.Itoa(s.idle.seconds/60) + " min idle)")
				if err != nil {
					fmt.Println("🔴 Error sending notification", err)
				}
				s.notified.idle = true
			}
		}

		time.Sleep(time.Duration(pollTimeSeconds) * time.Second)
	}
}
