package btcbuilder

import (
	"fmt"

	"github.com/conformal/btcscript"
	"github.com/conformal/btcwire"
)

type FanOutBuilder struct {
	Params   BuilderParams
	Builders []TxBuilder
	Copies   int64 // Number of copies to add
}

// A FanOutBuilder creates a transaction that has txouts set to the needed value
// for other tx builders that need those txouts as inputs
// The number of outputs created is len(builders)*copies + 1
func NewFanOutBuilder(params BuilderParams, builders []TxBuilder, copies int) *FanOutBuilder {
	fb := FanOutBuilder{
		Params:   params,
		Builders: builders,
		Copies:   int64(copies),
	}
	return &fb
}

func (fanB *FanOutBuilder) SatNeeded() int64 {
	sum := int64(0)
	for _, builder := range fanB.Builders {
		sum += builder.SatNeeded() * fanB.Copies
	}
	// Good Citizens pay the toll
	sum += fanB.Params.Fee
	return sum
}

func (fanB *FanOutBuilder) Build() (*btcwire.MsgTx, error) {
	totalSpent := fanB.SatNeeded()

	// Compose a set of Txins with enough to fund this transactions needs
	inParamSet, totalIn, err := composeUnspents(
		totalSpent,
		fanB.Params)
	if err != nil {
		return nil, err
	}

	msgtx := btcwire.NewMsgTx()
	// funding inputs speced out with blank
	for _, inpParam := range inParamSet {
		txin := btcwire.NewTxIn(inpParam.OutPoint, []byte{})
		msgtx.AddTxIn(txin)
	}

	for i := range fanB.Builders {
		builder := fanB.Builders[i]
		amnt := builder.SatNeeded()
		for j := int64(0); j < fanB.Copies; j++ {
			addr, err := newAddr(fanB.Params.Client)
			if err != nil {
				return nil, err
			}
			script, _ := btcscript.PayToAddrScript(addr)
			txout := btcwire.NewTxOut(amnt, script)
			msgtx.AddTxOut(txout)
		}
	}

	changeAddr, err := newAddr(fanB.Params.Client)
	if err != nil {
		return nil, err
	}
	// change to solve unevenness
	change, ok := changeOutput(totalIn-totalSpent, fanB.Params.DustAmnt, changeAddr)
	if ok {
		msgtx.AddTxOut(change)
	}

	// sign msgtx for each input
	for i, inpParam := range inParamSet {
		privkey := inpParam.Wif.PrivKey.ToECDSA()
		subscript := inpParam.TxOut.PkScript
		var sigflag byte
		sigflag = btcscript.SigHashAll
		scriptSig, err := btcscript.SignatureScript(msgtx, i, subscript,
			sigflag, privkey, true)
		if err != nil {
			return nil, err
		}
		msgtx.TxIn[i].SignatureScript = scriptSig
	}
	fanB.Log(fmt.Sprintf("InVal: %d\n", sumInputs(inParamSet)))
	fanB.Log(fmt.Sprintf("OutVal: %d\n", sumOutputs(msgtx)))

	return msgtx, nil
}

func (fanB *FanOutBuilder) Log(msg string) {
	fanB.Params.Logger.Println(msg)
}

func (fanB *FanOutBuilder) Summarize() string {
	s := "==== Fanout Tx ====\nSatNeeded:\t%d\nTxIns:\t?\nTxOuts:\t%d\n"
	s = fmt.Sprintf(s, fanB.SatNeeded(), int(fanB.Copies)*len(fanB.Builders))
	for _, builder := range fanB.Builders {
		s = s + builder.Summarize()
	}
	return s
}
