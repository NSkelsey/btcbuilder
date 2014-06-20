package btcbuilder

import (
	"log"
	"testing"

	"github.com/conformal/btcutil"
)

func TestRpc(t *testing.T) {
	log.Println("Testing to see if rpc config works for this node!")

	connCfg, _, err := CfgFromFile()
	if err != nil {
		log.Println(err)
		t.Fail()
	}

	client, err := makeRpcClient(connCfg)
	if err != nil {
		log.Println(err)
		t.Fail()
	}

	if err := checkconnection(client); err != nil {
		log.Println(err)
		t.Fail()
	}
}

func TestBalance(t *testing.T) {
	log.Println("Testing to see if wallet has adequate balance")
	client, _ := ConfigureApp()

	bal, err := client.GetBalance("")
	if err != nil {
		log.Println(err)
		t.Fail()
	}

	target, _ := btcutil.NewAmount(1) // one bitcoin balance targeted
	if bal < target {
		log.Printf("Not enough funds %s short\n", target-bal)
		t.Fail()
	}
}
