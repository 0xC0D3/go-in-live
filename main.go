package main

import "flag"
import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/nsf/termbox-go"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

/*
 * Some goals to reach:
 *
 * [x] - We need a param that receives the file/dir to watch for changes.
 * [x] - Must be capable to receive one or more input files/dir.
 * [x] - By default it will create a .c0d3v file that will be updated just pressing a key (maybe <ENTER>).
 * [ ] - Must be capable to pass live params to the executable (maybe to the compiler also).
 * [ ] - Maybe could run the tests before build/execute and only proceed if passes those tests.
 */

const (
	currentVersion = "0.0.1"
	c0d3vFilePath  = "./.c0d3v"
	exeFile        = "_test.exe"
)

var (
	cmdBuild, cmdRun string
	filesList        string
	redirectInput    bool
	watcher          *fsnotify.Watcher
	watchersPaths    []string
	c0d3vFile        *os.File
	runProcess       *os.Process
)

func init() {
	flag.StringVar(&filesList, "watch", c0d3vFilePath, "Comma separated paths to watch, in case you want to watch all files \ninside a directory, use the \".\\< dir >/*\" format.")
	flag.StringVar(&cmdBuild, "build", "go build -o $1", "Custom build command.")
	flag.StringVar(&cmdRun, "run", "$1", "Custom run command.")
	flag.BoolVar(&redirectInput, "i", false, "Redirect input to the executable.")

	flag.Parse()

	if flag.Arg(0) == "version" {
		fmt.Printf("go-in-live version %v", currentVersion)
		os.Exit(0)
	}

	cmdBuild = strings.Replace(cmdBuild, "$1", exeFile, -1)
	cmdRun = strings.Replace(cmdRun, "$1", exeFile, -1)

	fmt.Println(cmdBuild)
}

func main() {
	defer termbox.Close()

	var err error
	watchersPaths = strings.Split(filesList, ",")
	//fmt.Printf("%#v\n", watchersPaths)

	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	defer watcher.Close()

	initWatcher()

	waitMe := &sync.WaitGroup{}
	waitMe.Add(1) // Just for the consoleEventsLoop

	err = termbox.Init()
	if err != nil {
		log.Panic(err)
	}

	termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse)
	termbox.Flush()
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	fList := "\t\t\t- " + strings.Join(watchersPaths, "\n\t\t\t- ")
	log.Println(strings.Join(append([]string{"Watching files:", fList}), "\n"))

	go errPrinter(watcher.Errors)
	go filesWatcherEventsProcessor(watcher.Events)
	go consoleEventsLoop(waitMe)
	waitMe.Wait()

	errs := shutDown()

	if len(errs) > 0 {
		log.Println("Some errors ocurres during shutting down.")

		for _, err := range errs {
			log.Println(err)
		}
	}

	os.Exit(0)
}

func initWatcher() {
	var err error
	for _, path := range watchersPaths {
		if path == c0d3vFilePath {
			c0d3vFile, err = os.OpenFile(c0d3vFilePath, os.O_WRONLY+os.O_CREATE, 655)
			if err != nil {
				log.Panic(err)
			}
		}

		watcher.Add(path)
	}
}

func shutDown() []error {
	var errs = *new([]error)

	if runProcess != nil {
		if err := runProcess.Kill(); err != nil {
			errs = append(errs, errors.New(fmt.Sprintf("error killing process\n%v", err)))
		}

		runProcess.Wait()
	}

	if _, err := os.Stat(c0d3vFilePath); err == nil {
		if err := c0d3vFile.Close(); err != nil {
			errs = append(errs, errors.New(fmt.Sprintf("error closing file\n%v", err)))
		}
		if err := os.Remove(c0d3vFilePath); err != nil {
			errs = append(errs, errors.New(fmt.Sprintf("error deleting %v file\n%v", c0d3vFilePath, err)))
		}
	}

	if _, err := os.Stat(exeFile); err == nil {
		if err := os.Remove(exeFile); err != nil {
			errs = append(errs, errors.New(fmt.Sprintf("error deleting %v file\n%v", exeFile, err)))
		}
	}

	return errs
}

func errPrinter(errChan chan error) {
	for e := range errChan {
		log.Println(e.Error())
	}
}

func filesWatcherEventsProcessor(eventChan chan fsnotify.Event) {
	for ev := range eventChan {
		switch ev.Op {
		//case fsnotify.Write:
		default:
			log.Printf("File event [%v]", ev)
			buildAndRun()
			watcher.Remove(ev.Name)
			initWatcher()
		}
	}
}

func consoleEventsLoop(group *sync.WaitGroup) {
loop:
	for {
		ev := termbox.PollEvent()

		switch ev.Type {
		case termbox.EventKey:
			if keyEvent(ev) {
				break loop
			}

		case termbox.EventError:
			log.Panic(ev.Err)
		}
	}
	group.Done()
}

func keyEvent(ev termbox.Event) bool {
	switch ev.Key {
	case termbox.KeyF5:
		log.Println("Screen resync...")
		termbox.Sync()
	case termbox.KeyCtrlQ:
		log.Println("Quit!")
		return true
	case termbox.KeyCtrlB:
		log.Println("Build the project.")
		err := build()
		if err != nil {
			log.Panic(err)
		}
	case termbox.KeyCtrlR:
		var err error
		log.Println("Run the executable.")

		runProcess, err = run()
		if err != nil {
			log.Panic(err)
		}
	case termbox.KeyCtrlA:
		log.Println("Build & Run.")
		err := buildAndRun()
		if err != nil {
			log.Panic(err)
		}
	}

	return false
}

func buildAndRun() error {
	err := build()
	if err != nil {
		return err
	}

	runProcess, err = run()
	if err != nil {
		return err
	}

	return nil
}

func build() error {
	log.Printf("ex. %v\n", cmdBuild)
	exeLine := strings.Split(cmdBuild, " ")
	cmd := new(exec.Cmd)
	cmd = exec.Command(exeLine[0], exeLine[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func run() (*os.Process, error) {
	if runProcess != nil {
		runProcess.Kill()
	}
	log.Printf("ex. %v\n", cmdRun)
	exeLine := strings.Split(cmdRun, " ")
	cmd := new(exec.Cmd)
	cmd = exec.Command(exeLine[0], exeLine[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	if redirectInput {
		cmd.Stdin = os.Stdin
	}

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd.Process, nil
}
