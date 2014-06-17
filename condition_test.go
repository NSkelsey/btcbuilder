package btcbuilder

import (
	"log"
	"testing"

	"github.com/conformal/btcwire"
)

func TestBalance(t *testing.T) {
	client, _ := SetupNet(btcwire.TestNet3)
	amnt, _ := client.GetBalance("")

	balance := int64(amnt)

	if balance < 20000000 {
		log.Println("Not enough bitcoin in wallet: ", balance)
		t.Fail()
	}
}
