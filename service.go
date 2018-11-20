package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/hashicorp/consul/api"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Config struct {
	host  string
	port  int
	from  string
	to    string
	debug bool
}

type Task struct {
	closed chan struct{}
	wg     sync.WaitGroup
	ticker *time.Ticker
	config Config
}

func save(filepath string, s string) error {
	fo, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer fo.Close()

	_, err = io.Copy(fo, strings.NewReader(s))
	if err != nil {
		return err
	}

	log.Printf("[INFO] Saved: %s", filepath)

	return nil
}

func getHash(v []byte) string {
	hasher := sha256.New()
	hasher.Write(v)
	cksum := hex.EncodeToString(hasher.Sum(nil))

	return cksum
}

func calculateFileHash(filepath string) (string, bool) {
	f, err := ioutil.ReadFile(filepath)

	if err != nil {
		return "", false
	}

	return getHash(f), true
}

func getClient(host string, port int) *api.Client {
	config := api.DefaultConfig()
	config.Address = fmt.Sprintf("%s:%d", host, port)
	client, err := api.NewClient(config)
	if err != nil {
		log.Fatal(err.Error())
	}

	return client
}

func getConfig(host string, port int, from string, to string, debug bool) *Config {
	config := &Config{
		host:  host,
		port:  port,
		from:  from,
		to:    to,
		debug: debug,
	}

	return config
}

func process(config Config) {
	kv := getClient(config.host, config.port).KV()

	keys, _, err := kv.List(config.from, nil)
	if err != nil {
		log.Fatal(err)
	}

	if len(keys) <= 0 {
		log.Print("[WARNING] Consul path empty or does not exists")
	}

	for _, pair := range keys {
		parts    := strings.Split(pair.Key, "/")
		filename := parts[len(parts)-1]
		if filename != "" {
			file     := filepath.Join(config.to, filename)
			fHash, _ := calculateFileHash(file)
			sHash    := getHash(pair.Value[:])

			if fHash != sHash {
				if err := save(file, string(pair.Value[:])); err != nil {
					log.Fatal(err)
				}
			} else {
				if config.debug {
					log.Printf("[INFO] Same: %s", file)
				}
			}
		}
	}
}

func (t *Task) Run() {
	for {
		select {
			case <-t.closed:
				return
			case <-t.ticker.C:
				process(t.config)
		}
	}
}

func (t *Task) Stop() {
	close(t.closed)
	t.wg.Wait()
}

func main() {
	fmt.Print("[INFO] SignNow Consul Keys Creator\n\n")

	from    := flag.String("from", "nil", "Consul keys path") //"us-east-1/apps/dev-core-user-api/dev/keys/"
	to      := flag.String("to", "./", "Local path for keys")
	host    := flag.String("host", "consul", "Consul host")
	port    := flag.Int("port", 8500, "Consul port")
	seconds := flag.Int("seconds", 5, "Refresh interval")
	debug   := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	if _, err := os.Stat(*to); os.IsNotExist(err) {
		log.Print("[INFO] Destination folder does not exists. Creating...\n")
		os.MkdirAll(*to, os.ModePerm)
	}

	destination := strings.TrimSuffix(*to, "/")
	config      := getConfig(*host, *port, *from, destination, *debug)
	task        := &Task{
					closed: make(chan struct{}),
					ticker: time.NewTicker(time.Second * time.Duration(*seconds)),
					config: *config,
				}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	task.wg.Add(1)
	go func() { defer task.wg.Done(); task.Run() }()

	select {
		case sig := <-c:
			log.Printf("[WARNING] Got %s signal. Aborting...\n", sig)
			task.Stop()
	}
}
