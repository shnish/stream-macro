package streammacro

import (
	"io/ioutil"
	"fmt"
	"strings"
	"strconv"
	"github.com/go-vgo/robotgo"
	"github.com/google/gops/goprocess"
	"github.com/fsnotify/fsnotify"
)

// Map of game -> map of tip -> action string
var gameMap = make(map[string]map[int]string)
// TODO: Use for faster process finding rather than searching all processes??
var currentGame string

func configSetup() error {
	// Configuration files:
	files, err := ioutil.ReadDir("./")
	if err != nil {
		fmt.Printf("Failed to find configs in directory")
		return err
	}

	// Read all config files.  Looking for ".config" files
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".config") {
			continue
		}
		if bytearr, err := ioutil.ReadFile(file.Name()); err != nil {
			fmt.Printf("Error occurred while trying to read file %s. Error msg: %s", file.Name(), err)
		} else {
			var keyMap = make(map[int]string)
			var gameName string
			filestr := string(bytearr)
			str := strings.Split(filestr, "\n")
			for i, s := range str {
				switch i {
				case 0:
					// name of exe
					gameName = s
				default:
					// mapping of tip -> action
					tmp := strings.Split(s, ":")
					if value, err := strconv.ParseInt(tmp[0], 10, 32); err != nil {
						fmt.Printf("Error while trying to parse config line \"%s\". Error: %s... Skipping line", s, err)
					} else {
						keyMap[int(value)] = tmp[1]
					}
				}
			}
			gameMap[gameName] = keyMap
		}
	}
	return nil
}

func doAction(configAction string) error {
	actions := strings.Split(configAction, "|")
	for _, action := range actions {
		if len(action) < 2 {
			fmt.Printf("incorrect action string: %s", action)
			continue
		}
		a := []byte(strings.TrimSpace(action))
		switch a[0] {
		case 'k':
			// TODO: support click and hold / multi-char buttons
			robotgo.KeyTap(string(a[1]))
		case 'm':
			// TODO: support extra mouse buttons
			switch a[1] {
			case 'l':
				robotgo.MouseClick("left", false)
			case 'r':
				robotgo.MouseClick("right", false)
			default:
				// unmapped mouse action
				fmt.Printf("Failed to do mouse action %s in %s.", action, configAction)
			}
		default:
			// unknown action
			fmt.Printf("Failed to do action %s in %s.", action, configAction)
		}
	}
	return nil
}

func whichGameRunning() string {
	processes := goprocess.FindAll()
	for _, process := range processes {
		if gameMap[process.Exec] != nil {
			return process.Exec
		}
	}
	return ""
}

func main() {
	if err := configSetup(); err != nil {
		return
	}

	// creates a new file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("ERROR", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			// watch for events
			case event := <-watcher.Events:
				switch event.Op {
				case fsnotify.Write:
					if bytearr, err := ioutil.ReadFile(event.Name); err != nil {
						fmt.Printf("Error occurred while trying to read file %s. Error msg: %s", event.Name, err)
					} else {
						// Format: "user: amount"
						filestr := string(bytearr)
						str := strings.Split(filestr, ";")
						//user := str[0] // MIGHT BE USED LATER
						if amount, err := strconv.ParseInt( strings.TrimSpace(str[1]), 10, 32); err != nil {
							fmt.Printf("Failed to parse tip amount: %s", str[1])
						} else {
							game := whichGameRunning()
							if gameMap[game] != nil {
								doAction(gameMap[game][int(amount)])
							} else {
								fmt.Printf("No game running")
							}
						}
					}
				}
			// watch for errors
			case err := <-watcher.Errors:
				fmt.Println("ERROR", err)
			}
		}
	}()

	// out of the box fsnotify can watch a single file, or a single directory
	if err := watcher.Add("/Users/skdomino/Desktop/test.html"); err != nil {
		fmt.Println("ERROR", err)
	}

	<-done
}