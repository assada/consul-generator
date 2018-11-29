package manager

import (
	"fmt"
	"log"
	"os"
	"testing"

	dep "github.com/Assada/consul-generator/client"
	"github.com/hashicorp/consul/testutil"
)

var testConsul *testutil.TestServer
var testClients *dep.ClientSet

func TestMain(m *testing.M) {
	consul, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"
	})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to start consul server: %v", err))
	}
	testConsul = consul

	clients := dep.NewClientSet()
	if err := clients.CreateConsulClient(&dep.CreateConsulClientInput{
		Address: testConsul.HTTPAddr,
	}); err != nil {
		testConsul.Stop()
		log.Fatal(err)
	}
	testClients = clients

	exitCh := make(chan int, 1)
	func() {
		defer func() {
			if r := recover(); r != nil {
				testConsul.Stop()
				panic(r)
			}
		}()

		exitCh <- m.Run()
	}()

	exit := <-exitCh

	testConsul.Stop()
	os.Exit(exit)
}
