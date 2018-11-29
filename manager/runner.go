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

type Runner struct {
	ErrCh                chan error
	DoneCh               chan bool
	ticker               *time.Ticker
	config               *config.Config
	dry, once            bool
	outStream, errStream io.Writer
	inStream             io.Reader
	stopLock             sync.Mutex
	stopped              bool
}

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

func (r *Runner) Start() {
	log.Printf("[INFO] (runner) starting")

	if err := r.storePid(); err != nil {
		r.ErrCh <- err
		return
	}

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

func (r *Runner) Run() error {
	log.Printf("[DEBUG] (runner) initiating run")

	return nil
}

func (r *Runner) init() error {
	r.config = config.DefaultConfig().Merge(r.config)
	r.config.Finalize()

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

func (r *Runner) SetOutStream(out io.Writer) {
	r.outStream = out
}

func (r *Runner) SetErrStream(err io.Writer) {
	r.errStream = err
}
