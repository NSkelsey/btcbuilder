package btcbuilder

import (
	"log"
	"testing"

	"github.com/conformal/btcwire"
	_ "gopkg.in/check.v1"
)

func TestRpcParameters(t *testing.T) {
	log.Println("Testing to see if rpc config works for this node!")

	// TODO add parsing of config ini file
	// from https://github.com/jessevdk/go-flags/blob/master/examples/main.go
	client, err := SetupNet(btcwire.TestNet3)
	if err != nil {
		log.Println(err)
		t.Fail()
	}

	info, err := client.GetInfo()
	if err != nil {
		log.Println(err)
		t.Fail()
	}

}
