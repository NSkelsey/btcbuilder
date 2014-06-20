package btcbuilder

import (
	"log"
	"os"
	"sort"

	"github.com/conformal/btcjson"
	"github.com/conformal/btcnet"
	"github.com/conformal/btcrpcclient"
	"github.com/conformal/btcscript"
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

func SetParams(net btcwire.BitcoinNet, params BuilderParams) BuilderParams {
	if params.Logger == nil {
		params.Logger = log.New(os.Stdout, "", log.Ltime|log.Llongfile)
	}
	if params.Client == nil {
		client, currnet := ConfigureApp()
		params.Client = client
		params.NetParams = &currnet
		params.PendingSet = make(map[string]struct{})
		params.List = make([]btcjson.ListUnspentResult, 0)
	}

	return params
}

// TODO combine entry points into library into one global configuration function
func CreateParams() BuilderParams {
	var logger *log.Logger = log.New(os.Stdout, "", log.Ltime|log.Llongfile)
	client, params := ConfigureApp()

	bp := BuilderParams{
		Fee:        20000,
		DustAmnt:   546,
		InTarget:   100000,
		Logger:     logger,
		Client:     client,
		NetParams:  &params,
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

// extracts the counts of the standard outscripts each transaction contains
func ExtractOutScripts(tx *btcwire.MsgTx) map[btcscript.ScriptClass]int {
	outmap := make(map[btcscript.ScriptClass]int)
	for _, txout := range tx.TxOut {
		class := btcscript.GetScriptClass(txout.PkScript)
		outmap[class]++
	}
	return outmap
}

type Pair struct {
	Num   int
	Class btcscript.ScriptClass
}

type PairList []Pair

func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Num < p[j].Num }

// SelectKind picks from a set of known tx types a transactions `kind`
// Which is the set of enumerated transaction we can identify based on
// the properties of that transaction.
func SelectKind(tx *btcwire.MsgTx) string {
	counts := ExtractOutScripts(tx)
	if len(counts) < 1 {
		return "nonstandard"
	}

	pl := make(PairList, 0)
	for cls, num := range counts {
		switch {
		case cls == btcscript.NonStandardTy:
			return "nonstandard"
		case cls == btcscript.NullDataTy:
			return "nulldata"
		case cls == btcscript.MultiSigTy:
			return "multisig"
		}
		pl = append(pl, Pair{Num: num, Class: cls})
	}
	// If the does not have funky output scripts just count occurrences
	sort.Sort(pl)
	return pl[0].Class.String()
}
