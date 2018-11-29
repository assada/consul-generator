package manager

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Assada/consul-generator/child"
	"github.com/Assada/consul-generator/config"
	"github.com/Assada/consul-generator/processor"

	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
)

const (
	// saneViewLimit is the number of views that we consider "sane" before we
	// warn the user that they might be DDoSing their Consul cluster.
	saneViewLimit = 128
)

// Runner responsible rendering Templates and invoking Commands.
type Runner struct {
	//sync.Mutex
	// ErrCh and DoneCh are channels where errors and finish notifications occur.
	ErrCh  chan error
	DoneCh chan bool

	ticker *time.Ticker

	// config is the Config that created this Runner. It is used internally to
	// construct other objects and pass data.
	config *config.Config

	// dry signals that output should be sent to stdout instead of committed to
	// disk. once indicates the runner should execute each template exactly one
	// time and then stop.
	dry, once bool

	// outStream and errStream are the io.Writer streams where the runner will
	// write information. These can be modified by calling SetOutStream and
	// SetErrStream accordingly.

	// inStream is the ioReader where the runner will read information.
	outStream, errStream io.Writer
	inStream             io.Reader

	// renderEvents is a mapping of a template ID to the render event.
	renderEvents map[string]*RenderEvent

	// renderEventLock protects access into the renderEvents map
	renderEventsLock sync.RWMutex

	// renderedCh is used to signal that a template has been rendered
	renderedCh chan struct{}

	// renderEventCh is used to signal that there is a new render event. A
	// render event doesn't necessarily mean that a template has been rendered,
	// only that templates attempted to render and may have updated their
	// dependency sets.
	renderEventCh chan struct{}

	// dependenciesLock is a lock around touching the dependencies map.
	dependenciesLock sync.Mutex

	// child is the child process under management. This may be nil if not running
	// in exec mode.
	child *child.Child

	// childLock is the internal lock around the child process.
	childLock sync.RWMutex

	// quiescenceMap is the map of templates to their quiescence timers.
	// quiescenceCh is the channel where templates report returns from quiescence
	// fires.
	quiescenceMap map[string]*quiescence

	// Env represents a custom set of environment variables to populate the
	// template and command runtime with. These environment variables will be
	// available in both the command's environment as well as the template's
	// environment.
	Env map[string]string

	// stopLock is the lock around checking if the runner can be stopped
	stopLock sync.Mutex

	// stopped is a boolean of whether the runner is stopped
	stopped bool
}

// RenderEvent captures the time and events that occurred for a template
// rendering.
type RenderEvent struct {
	// Contents is the raw, rendered contents from the template.
	Contents []byte

	// UpdatedAt is the last time this render event was updated.
	UpdatedAt time.Time

	// WouldRender determines if the template would have been rendered. A template
	// would have been rendered if all the dependencies are satisfied, but may
	// not have actually rendered if the file was already present or if an error
	// occurred when trying to write the file.
	WouldRender bool

	// LastWouldRender marks the last time the template would have rendered.
	LastWouldRender time.Time

	// DidRender determines if the Template was actually written to disk. In dry
	// mode, this will always be false, since templates are not written to disk
	// in dry mode. A template is only rendered to disk if all dependencies are
	// satisfied and the template is not already in place with the same contents.
	DidRender bool

	// LastDidRender marks the last time the template was written to disk.
	LastDidRender time.Time
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
	r.stopChild()

	if err := r.deletePid(); err != nil {
		log.Printf("[WARN] (runner) could not remove pid at %q: %s",
			r.config.PidFile, err)
	}

	r.stopped = true

	close(r.DoneCh)
}

// RenderEvents returns the render events for each template was rendered. The
// map is keyed by template ID.
func (r *Runner) RenderEvents() map[string]*RenderEvent {
	r.renderEventsLock.RLock()
	defer r.renderEventsLock.RUnlock()

	times := make(map[string]*RenderEvent, len(r.renderEvents))
	for k, v := range r.renderEvents {
		times[k] = v
	}
	return times
}

func (r *Runner) stopChild() {
	r.childLock.RLock()
	defer r.childLock.RUnlock()

	if r.child != nil {
		log.Printf("[DEBUG] (runner) stopping child process")
		r.child.Stop()
	}
}

// Signal sends a signal to the child process, if it exists. Any errors that
// occur are returned.
func (r *Runner) Signal(s os.Signal) error {
	r.childLock.RLock()
	defer r.childLock.RUnlock()
	if r.child == nil {
		return nil
	}
	return r.child.Signal(s)
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

	r.renderEvents = make(map[string]*RenderEvent, 2)

	r.renderedCh = make(chan struct{}, 1)
	r.renderEventCh = make(chan struct{}, 1)

	r.inStream = os.Stdin
	r.outStream = os.Stdout
	r.errStream = os.Stderr

	r.ErrCh = make(chan error)
	r.DoneCh = make(chan bool)

	r.quiescenceMap = make(map[string]*quiescence)

	return nil
}

// childEnv creates a map of environment variables for child processes to have
// access to configurations in Consul Template's configuration.
func (r *Runner) childEnv() []string {
	var m = make(map[string]string)

	if config.StringPresent(r.config.Consul.Address) {
		m["CONSUL_HTTP_ADDR"] = config.StringVal(r.config.Consul.Address)
	}

	if config.BoolVal(r.config.Consul.Auth.Enabled) {
		m["CONSUL_HTTP_AUTH"] = r.config.Consul.Auth.String()
	}

	m["CONSUL_HTTP_SSL"] = strconv.FormatBool(config.BoolVal(r.config.Consul.SSL.Enabled))
	m["CONSUL_HTTP_SSL_VERIFY"] = strconv.FormatBool(config.BoolVal(r.config.Consul.SSL.Verify))

	// Append runner-supplied env (this is supplied programmatically).
	for k, v := range r.Env {
		m[k] = v
	}

	e := make([]string, 0, len(m))
	for k, v := range m {
		e = append(e, k+"="+v)
	}
	return e
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

// spawnChildInput is used as input to spawn a child process.
type spawnChildInput struct {
	Stdin        io.Reader
	Stdout       io.Writer
	Stderr       io.Writer
	Command      string
	Timeout      time.Duration
	Env          []string
	ReloadSignal os.Signal
	KillSignal   os.Signal
	KillTimeout  time.Duration
	Splay        time.Duration
}

// spawnChild spawns a child process with the given inputs and returns the
// resulting child.
func spawnChild(i *spawnChildInput) (*child.Child, error) {
	p := shellwords.NewParser()
	p.ParseEnv = true
	p.ParseBacktick = true
	args, err := p.Parse(i.Command)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing command")
	}

	child, err := child.New(&child.NewInput{
		Stdin:        i.Stdin,
		Stdout:       i.Stdout,
		Stderr:       i.Stderr,
		Command:      args[0],
		Args:         args[1:],
		Env:          i.Env,
		Timeout:      i.Timeout,
		ReloadSignal: i.ReloadSignal,
		KillSignal:   i.KillSignal,
		KillTimeout:  i.KillTimeout,
		Splay:        i.Splay,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error creating child")
	}

	if err := child.Start(); err != nil {
		return nil, errors.Wrap(err, "child")
	}
	return child, nil
}

// quiescence is an internal representation of a single template's quiescence
// state.
type quiescence struct {
	//template *template.Template
	min time.Duration
	max time.Duration
	//ch       chan *template.Template
	timer    *time.Timer
	deadline time.Time
}
