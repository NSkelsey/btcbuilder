package btcbuilder

import (
	"log"
	"os"

	"github.com/conformal/btcjson"
	"github.com/conformal/btcnet"
	"github.com/conformal/btcrpcclient"
	"github.com/conformal/btcwire"
)

type BuilderParams struct {
	Fee        int64
	DustAmnt   int64
	InTarget   int64 // The target input a transaction must be created with
	Logger     *log.Logger
	Client     *btcrpcclient.Client
	NetParams  *btcnet.Params
	PendingSet map[string]struct{}
	List       []btcjson.ListUnspentResult
}

type TxBuilder interface {
	// SatNeeded computes the specific value needed at an txout for the tx being built by the builder
	SatNeeded() int64
	// Build generates a MsgTx from the provided parameters, (rpc client, FEE, ...)
	Build() (*btcwire.MsgTx, error)
	// Log is short hand for logging in a tx builder with Param logger
	Log(string)
	Summarize() string
}

func CreateParams() BuilderParams {
	var logger *log.Logger = log.New(os.Stdout, "", log.Ltime|log.Llongfile)
	client, params := SetupNet(btcwire.TestNet3)

	bp := BuilderParams{
		Fee:        20000,
		DustAmnt:   546,
		InTarget:   100000,
		Logger:     logger,
		Client:     client,
		NetParams:  params,
		PendingSet: make(map[string]struct{}),
		List:       make([]btcjson.ListUnspentResult, 0),
	}
	return bp
}
func Send(builder TxBuilder, params BuilderParams) *btcwire.ShaHash {
	msg, err := builder.Build()
	if err != nil {
		log.Fatal(err)
	}
	println(ToHex(msg))
	resp, err := params.Client.SendRawTransaction(msg, false)
	if err != nil {
		log.Fatal(err)
	}
	return resp
}
