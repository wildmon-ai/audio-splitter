// Command audio-splitter-backend cuts fixed-length clips out of long audio
// recordings with ffmpeg. The Electron app drives it over stdio:
//
//	stdin:  one JSON command per line: {"cmd":"start","config":{...}} or {"cmd":"cancel"}
//	stdout: one JSON event per line: "log", "progress" and "done"
//
// Every accepted start command ends with exactly one done event. Closing stdin
// (or SIGINT/SIGTERM) cancels the running job, stops its ffmpeg processes and
// exits.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"unicode"
	"unicode/utf8"
)

func main() {
	events := newEmitter(os.Stdout)
	jobs := &jobRunner{events: events}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		jobs.shutdown()
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var cmd struct {
			Cmd    string          `json:"cmd"`
			Config json.RawMessage `json:"config"`
		}
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			events.log(levelError, "Ignored a malformed command: "+err.Error())
			continue
		}
		switch cmd.Cmd {
		case "start":
			jobs.start(cmd.Config)
		case "cancel":
			jobs.cancel()
		default:
			events.log(levelError, "Ignored an unknown command: "+cmd.Cmd)
		}
	}
	// stdin closed: the app is quitting.
	jobs.shutdown()
}

// jobRunner runs at most one job at a time.
type jobRunner struct {
	events *emitter

	mu        sync.Mutex
	cancelJob context.CancelFunc // set while a job runs
	closed    bool
	wg        sync.WaitGroup
}

func (j *jobRunner) start(rawConfig json.RawMessage) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return
	}
	if j.cancelJob != nil {
		j.events.log(levelError, "A job is already running.")
		return
	}
	cfg, err := parseConfig(rawConfig)
	if err != nil {
		j.events.send(j.events.fail(err))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	j.cancelJob = cancel
	j.wg.Add(1)
	go func() {
		defer j.wg.Done()
		result := run(ctx, cfg, findTools(), j.events)
		j.mu.Lock()
		j.cancelJob = nil
		j.mu.Unlock()
		cancel()
		// Sent after the job is cleared, so the app can start the next one
		// as soon as it sees this event.
		j.events.send(result)
	}()
}

func (j *jobRunner) cancel() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.cancelJob != nil {
		j.cancelJob()
	}
}

// shutdown cancels the running job and waits until its ffmpeg processes are gone.
func (j *jobRunner) shutdown() {
	j.mu.Lock()
	j.closed = true
	if j.cancelJob != nil {
		j.cancelJob()
	}
	j.mu.Unlock()
	j.wg.Wait()
}

type level string

const (
	levelInfo  level = "info"
	levelWarn  level = "warn"
	levelError level = "error"
)

type logEvent struct {
	Type    string `json:"type"`
	Level   level  `json:"level"`
	Message string `json:"message"`
}

type progressEvent struct {
	Type    string `json:"type"`
	Phase   string `json:"phase"` // "scan" while reading durations, then "clips"
	Current int    `json:"current"`
	Total   int    `json:"total"`
}

type doneEvent struct {
	Type       string `json:"type"`
	TotalClips int    `json:"total_clips"`
	Skipped    int    `json:"skipped"`
	Errors     int    `json:"errors"`
	Cancelled  bool   `json:"cancelled,omitempty"`
	Error      string `json:"error,omitempty"` // the job could not run at all
}

// emitter writes events as JSON lines. It is safe for concurrent use.
type emitter struct {
	mu sync.Mutex
	w  io.Writer
}

func newEmitter(w io.Writer) *emitter {
	return &emitter{w: w}
}

func (e *emitter) send(event any) {
	line, err := json.Marshal(event)
	if err != nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, _ = e.w.Write(append(line, '\n'))
}

func (e *emitter) log(l level, message string) {
	e.send(logEvent{Type: "log", Level: l, Message: message})
}

// fail logs an error that stops a job before any clip is cut and returns the
// done event that reports it.
func (e *emitter) fail(err error) doneEvent {
	message := sentence(err.Error())
	e.log(levelError, message)
	return doneEvent{Type: "done", Error: message}
}

// sentence capitalizes a Go error string for display.
func sentence(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}
