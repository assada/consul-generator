package processor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Assada/consul-generator/client"
	"github.com/Assada/consul-generator/config"
	"github.com/hashicorp/consul/api"
)

const (
	ExitCodeOK    int = 0
	ExitCodeError     = 10 + iota
)

type Processor struct {
	config config.Config
	kv     api.KV
	error  chan error
	done   chan bool
	once   bool
	dry    bool
}

func (p *Processor) save(filepath string, s string) error {
	if p.dry {
		log.Printf("File %s will be created with content: \n %s", filepath, s)
		return nil
	}
	fo, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer fo.Close()

	_, err = io.Copy(fo, strings.NewReader(s))
	if err != nil {
		return err
	}

	log.Printf("[INFO] (processor) Saved: %s", filepath)

	return nil
}

func (p *Processor) getHash(v []byte) string {
	hasher := sha256.New()
	hasher.Write(v)
	cksum := hex.EncodeToString(hasher.Sum(nil))

	return cksum
}

func (p *Processor) calculateFileHash(filepath string) (string, error) {
	f, err := ioutil.ReadFile(filepath)

	if err != nil {
		return "", err
	}

	return p.getHash(f), nil
}

func NewProcessor(config *config.Config, once bool, dry bool, errorCh chan error, doneCh chan bool) (*Processor, error) {
	log.Printf("[INFO] (processor) creating new processor")

	cl, err := newClientSet(config)
	if err != nil {
		logError(err, ExitCodeError)
	}

	processor := &Processor{
		config: *config,
		kv:     *cl.Consul().KV(),
		error:  errorCh,
		done:   doneCh,
		once:   once,
		dry:    dry,
	}

	processor.init()

	return processor, nil
}

func (p *Processor) init() {

	if p.dry == false {
		if _, err := os.Stat(*p.config.To); os.IsNotExist(err) {
			log.Print("[INFO] (processor) Destination folder does not exists. Creating...\n")
			err := os.MkdirAll(*p.config.To, os.ModePerm)
			if err != nil {
				p.error <- err
				logError(err, ExitCodeError)
			}
		}
	} else {
		log.Print("Destination folder does not exists. It will be created\n")
	}

}

func logError(err error, status int) int {
	log.Printf("[ERR] (processor) %s", err)
	return status
}

func (p *Processor) Process() int {
	keys, _, err := p.kv.List(*p.config.From, nil)
	if err != nil {
		p.error <- err
		return logError(err, ExitCodeError)
	}

	if len(keys) <= 0 {
		log.Printf("[WARNING] (processor) Consul path (%s) empty or does not exists", *p.config.From)
	} else {
		log.Printf("[INFO] (processor) Consul Path: %s", *p.config.From)
	}

	for _, pair := range keys {
		parts := strings.Split(pair.Key, "/")
		filename := parts[len(parts)-1]
		if filename != "" {
			file := filepath.Join(*p.config.To, filename)
			fHash, _ := p.calculateFileHash(file)
			sHash := p.getHash(pair.Value[:])

			if fHash != sHash {
				if err := p.save(file, string(pair.Value[:])); err != nil {
					p.error <- err
					return logError(err, ExitCodeError)
				}
			} else {
				log.Printf("[INFO] (processor) Skipping: %s", pair.Key)
			}
		}
	}
	if p.once || p.dry {
		p.done <- true
	}

	return ExitCodeOK
}

// newClientSet creates a new client set from the given config.
func newClientSet(c *config.Config) (*client.ClientSet, error) {
	clients := client.NewClientSet()

	if err := clients.CreateConsulClient(&client.CreateConsulClientInput{
		Address:                      config.StringVal(c.Consul.Address),
		Token:                        config.StringVal(c.Consul.Token),
		AuthEnabled:                  config.BoolVal(c.Consul.Auth.Enabled),
		AuthUsername:                 config.StringVal(c.Consul.Auth.Username),
		AuthPassword:                 config.StringVal(c.Consul.Auth.Password),
		SSLEnabled:                   config.BoolVal(c.Consul.SSL.Enabled),
		SSLVerify:                    config.BoolVal(c.Consul.SSL.Verify),
		SSLCert:                      config.StringVal(c.Consul.SSL.Cert),
		SSLKey:                       config.StringVal(c.Consul.SSL.Key),
		SSLCACert:                    config.StringVal(c.Consul.SSL.CaCert),
		SSLCAPath:                    config.StringVal(c.Consul.SSL.CaPath),
		ServerName:                   config.StringVal(c.Consul.SSL.ServerName),
		TransportDialKeepAlive:       config.TimeDurationVal(c.Consul.Transport.DialKeepAlive),
		TransportDialTimeout:         config.TimeDurationVal(c.Consul.Transport.DialTimeout),
		TransportDisableKeepAlives:   config.BoolVal(c.Consul.Transport.DisableKeepAlives),
		TransportIdleConnTimeout:     config.TimeDurationVal(c.Consul.Transport.IdleConnTimeout),
		TransportMaxIdleConns:        config.IntVal(c.Consul.Transport.MaxIdleConns),
		TransportMaxIdleConnsPerHost: config.IntVal(c.Consul.Transport.MaxIdleConnsPerHost),
		TransportTLSHandshakeTimeout: config.TimeDurationVal(c.Consul.Transport.TLSHandshakeTimeout),
	}); err != nil {
		return nil, fmt.Errorf("runner: %s", err)
	}

	return clients, nil
}
