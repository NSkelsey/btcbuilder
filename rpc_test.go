package btcbuilder

import (
	"fmt"
	"testing"

	"log"

	"github.com/conformal/btcrpcclient"
	"github.com/davecgh/go-spew/spew"
)

func TestUnspent(t *testing.T) {
	connCfg := &btcrpcclient.ConnConfig{
		Host:         "localhost:18332",
		User:         "bitcoinrpc",
		Pass:         "EhxWGNKr1Z4LLqHtfwyQDemCRHF8gem843pnLj19K4go",
		DisableTLS:   true,
		HttpPostMode: true,
	}

	client, err := btcrpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	res, err := client.ListUnspent()
	if err != nil {
		bad := "json: cannot unmarshal object into Go value of type []btcjson.ListUnspentResult"
		if fmt.Sprintf("%s", err) == bad {
			log.Println("Died because unspent is too big")
			t.Fail()
		}
		log.Fatal(err)
	}
	log.Printf("Responded with %s", spew.Sdump(res))
}
