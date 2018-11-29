package manager

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Assada/consul-generator/config"
	"github.com/Assada/consul-generator/processor"
)

const (
	// saneViewLimit is the number of views that we consider "sane" before we
	// warn the user that they might be DDoSing their Consul cluster.
	saneViewLimit = 128
)

// Runner responsible rendering Templates and invoking Commands.
type Runner struct {
	ErrCh  chan error
	DoneCh chan bool

	ticker *time.Ticker

	config *config.Config

	dry, once bool

	outStream, errStream io.Writer
	inStream             io.Reader

	stopLock sync.Mutex

	stopped bool
}

// NewRunner accepts a slice of TemplateConfigs and returns a pointer to the new
// Runner and any error that occurred during creation.
func NewRunner(config *config.Config, dry, once bool) (*Runner, error) {
	log.Printf("[INFO] (runner) creating new runner (dry: %v, once: %v)", dry, once)

	runner := &Runner{
		config: config,
		dry:    dry,
		once:   once,
		ticker: time.NewTicker(*config.Interval),
	}

	if err := runner.init(); err != nil {
		return nil, err
	}

	return runner, nil
}

// Start begins the polling for this runner. Any errors that occur will cause
// this function to push an item onto the runner's error channel and the halt
// execution. This function is blocking and should be called as a goroutine.
func (r *Runner) Start() {
	log.Printf("[INFO] (runner) starting")

	// Create the pid before doing anything.
	if err := r.storePid(); err != nil {
		r.ErrCh <- err
		return
	}

	// Fire an initial run to parse all the templates and setup the first-pass
	// dependencies. This also forces any templates that have no dependencies to
	// be rendered immediately (since they are already renderable).
	log.Printf("[DEBUG] (runner) running initial templates")
	if err := r.Run(); err != nil {
		r.ErrCh <- err
		return
	}

	pr, _ := processor.NewProcessor(r.config, r.once, r.dry, r.ErrCh, r.DoneCh)

	for {
		select {
		case <-r.ErrCh:
			return
		case <-r.ticker.C:
			pr.Process()
		case <-r.DoneCh:
			log.Printf("[INFO] (runner) received finish")
			return
		}
	}

}

// Stop halts the execution of this runner and its subprocesses.
func (r *Runner) Stop() {
	r.stopLock.Lock()
	defer r.stopLock.Unlock()

	if r.stopped {
		return
	}

	log.Printf("[INFO] (runner) stopping")

	if err := r.deletePid(); err != nil {
		log.Printf("[WARN] (runner) could not remove pid at %q: %s",
			config.StringVal(r.config.PidFile), err)
	}

	r.stopped = true

	close(r.DoneCh)
}

// Run iterates over each template in this Runner and conditionally executes
// the template rendering and command execution.
//
// The template is rendered atomically. If and only if the template render
// completes successfully, the optional commands will be executed, if given.
// Please note that all templates are rendered **and then** any commands are
// executed.
func (r *Runner) Run() error {
	log.Printf("[DEBUG] (runner) initiating run")

	return nil
}

// init() creates the Runner's underlying data structures and returns an error
// if any problems occur.
func (r *Runner) init() error {
	// Ensure default configuration values
	r.config = config.DefaultConfig().Merge(r.config)
	r.config.Finalize()

	// Print the final config for debugging
	result, err := json.Marshal(r.config)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] (runner) final config: %s", result)

	r.inStream = os.Stdin
	r.outStream = os.Stdout
	r.errStream = os.Stderr

	r.ErrCh = make(chan error)
	r.DoneCh = make(chan bool)

	return nil
}

// storePid is used to write out a PID file to disk.
func (r *Runner) storePid() error {
	path := config.StringVal(r.config.PidFile)
	if path == "" {
		return nil
	}

	log.Printf("[INFO] creating pid file at %q", path)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("runner: could not open pid file: %s", err)
	}
	defer f.Close()

	pid := os.Getpid()
	_, err = f.WriteString(fmt.Sprintf("%d", pid))
	if err != nil {
		return fmt.Errorf("runner: could not write to pid file: %s", err)
	}
	return nil
}

// deletePid is used to remove the PID on exit.
func (r *Runner) deletePid() error {
	path := config.StringVal(r.config.PidFile)
	if path == "" {
		return nil
	}

	log.Printf("[DEBUG] removing pid file at %q", path)

	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("runner: could not remove pid file: %s", err)
	}
	if stat.IsDir() {
		return fmt.Errorf("runner: specified pid file path is directory")
	}

	err = os.Remove(path)
	if err != nil {
		return fmt.Errorf("runner: could not remove pid file: %s", err)
	}
	return nil
}

// SetOutStream modifies runner output stream. Defaults to stdout.
func (r *Runner) SetOutStream(out io.Writer) {
	r.outStream = out
}

// SetErrStream modifies runner error stream. Defaults to stderr.
func (r *Runner) SetErrStream(err io.Writer) {
	r.errStream = err
}
