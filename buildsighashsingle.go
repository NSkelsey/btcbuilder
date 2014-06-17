package btcbuilder

import (
	"errors"
	"fmt"

	"github.com/conformal/btcscript"
	"github.com/conformal/btcwire"
)

type SigHashSingleBuilder struct {
	Params BuilderParams
}

func NewSigHashSingleBuilder(params BuilderParams) *SigHashSingleBuilder {
	shsB := SigHashSingleBuilder{
		Params: params,
	}
	return &shsB
}

func (shsB *SigHashSingleBuilder) Build() (*btcwire.MsgTx, error) {
	// RPC to setup previous TX
	utxo, err := selectUnspent(shsB.SatNeeded()+shsB.Params.DustAmnt, shsB.Params)
	if err != nil {
		return nil, err
	}

	oldTxOut := utxo.TxOut
	outpoint := utxo.OutPoint
	wifkey := utxo.Wif

	// Transaction building

	txin := btcwire.NewTxIn(outpoint, []byte{})

	// notice amount in
	total := oldTxOut.Value
	changeval := total - (shsB.SatNeeded())
	change, ok := changeOutput(changeval, shsB.Params.DustAmnt,
		wifToAddr(wifkey, shsB.Params.NetParams))
	if !ok {
		return nil, errors.New("Not enough for change.")
	}
	// Blank permutable txout for users to play with
	blankval := shsB.Params.InTarget - shsB.Params.Fee
	blank := btcwire.NewTxOut(blankval, change.PkScript) //[]byte{})

	msgtx := btcwire.NewMsgTx()
	msgtx.AddTxIn(txin)
	msgtx.AddTxOut(change)
	msgtx.AddTxOut(blank)

	subscript := oldTxOut.PkScript
	privkey := wifkey.PrivKey.ToECDSA()
	scriptSig, err := btcscript.SignatureScript(msgtx, 0, subscript, btcscript.SigHashSingle, privkey, true)
	if err != nil {
		return nil, err
	}

	msgtx.TxIn[0].SignatureScript = scriptSig
	// This demonstrates that we can sign and then permute a txout
	//msgtx.TxOut[1].PkScript = oldTxOut.PkScript
	blank.Value = blankval + 1

	return msgtx, nil
}

func (shsB *SigHashSingleBuilder) SatNeeded() int64 {
	return shsB.Params.InTarget
}

func (shsB *SigHashSingleBuilder) Log(msg string) {
	shsB.Params.Logger.Println(msg)
}

func (shsB *SigHashSingleBuilder) Summarize() string {
	s := "==== SigHashSingle ====\nSatNeeded:\t%d\nTxIns:\t?\nTxOuts:\t2"
	return fmt.Sprintf(s, shsB.SatNeeded())
}
