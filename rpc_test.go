package btcbuilder

import (
	"fmt"
	"testing"

	"log"

	"github.com/conformal/btcnet"
	"github.com/conformal/btcrpcclient"
	"github.com/conformal/btcutil"
	"github.com/davecgh/go-spew/spew"
)

var connCfg *btcrpcclient.ConnConfig = &btcrpcclient.ConnConfig{
	Host:         "localhost:18332",
	User:         "bitcoinrpc",
	Pass:         "9uTysQtMLf15DGWDYcQVStEbWKcNu8CqCL8Mb6HE3xFK",
	HttpPostMode: true, // Bitcoin core only supports HTTP POST mode
	DisableTLS:   true, // Bitcoin core does not provide TLS by default
}

func TestBasic(t *testing.T) {
	log.Println("Testing Basic")
	// Connect to local bitcoin core RPC server using HTTP POST mode.
	// not supported in HTTP POST mode.
	log.Println("Connecting")
	client, err := btcrpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Shutdown()

	log.Println("Running getblockcount")
	// Get the current blocck count.
	blockCount, err := client.GetInfo()
	if err != nil {
		log.Println(err)
	}
	log.Printf("Block count: %d", blockCount)

}

func TestUnspent(t *testing.T) {
	log.Println("testing unspent")

	log.Println("Created client")
	client, err := btcrpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Testing unspent")

	net := btcnet.TestNet3Params
	addr, _ := btcutil.DecodeAddress("mmjN4Cs2KdHMeJgYBy6D75zbVhGKFh6t6H", &net)
	validate, err := client.ValidateAddress(addr)
	if err != nil {
		log.Println(err)
		log.Println(validate)
	}

	log.Println("Addr worked")

	res, err := client.ListUnspent()
	if err != nil {
		log.Println(err)
		bad := "json: cannot unmarshal object into Go value of type []btcjson.ListUnspentResult"
		if fmt.Sprintf("%s", err) == bad {
			t.Fail()
		}
	}
	log.Printf("Responded with %s", spew.Sdump(res))
}
