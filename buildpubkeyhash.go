package btcbuilder

import (
	"errors"
	"fmt"

	"github.com/conformal/btcscript"
	"github.com/conformal/btcwire"
)

type PubKeyHashBuilder struct {
	Params  BuilderParams
	NumOuts int64
}

func NewPayToPubKeyHash(params BuilderParams, numouts int64) *PubKeyHashBuilder {
	pkhB := PubKeyHashBuilder{
		Params:  params,
		NumOuts: numouts,
	}
	return &pkhB
}

// pkhbuilders are strict about there inTarget
// If num outs does not form a valid amount we round down
func (pkhB *PubKeyHashBuilder) SatNeeded() int64 {
	return pkhB.eachOutVal()*pkhB.NumOuts + pkhB.Params.Fee
}

// the amount sent to each output
func (pkhB *PubKeyHashBuilder) eachOutVal() int64 {
	n := pkhB.NumOuts
	t := pkhB.Params.InTarget
	f := pkhB.Params.Fee

	each := (t - f) / n
	return each
}

func (pkhB *PubKeyHashBuilder) Build() (*btcwire.MsgTx, error) {

	inparams, err := specificUnspent(pkhB.SatNeeded(), pkhB.Params)
	if err != nil {
		return nil, err
	}

	msgtx := btcwire.NewMsgTx()

	txin := btcwire.NewTxIn(inparams.OutPoint, []byte{})
	msgtx.AddTxIn(txin)

	for i := int64(0); i < pkhB.NumOuts; i++ {
		addr, err := newAddr(pkhB.Params.Client)
		if err != nil {
			return nil, err
		}
		addrScript, err := btcscript.PayToAddrScript(addr)
		amntSend := pkhB.eachOutVal()
		if amntSend < pkhB.Params.DustAmnt {
			return nil, errors.New("Output would be under the dust limit")
		}
		txout := btcwire.NewTxOut(pkhB.eachOutVal(), addrScript)
		msgtx.AddTxOut(txout)
	}
	privkey := inparams.Wif.PrivKey
	sig, err := btcscript.SignatureScript(msgtx,
		0,
		inparams.TxOut.PkScript,
		btcscript.SigHashAll,
		privkey,
		true)
	if err != nil {
		return nil, err
	}
	txin.SignatureScript = sig

	return msgtx, nil
}

func (pkhB *PubKeyHashBuilder) Log(msg string) {
	pkhB.Params.Logger.Println(msg)
}

func (pkhB *PubKeyHashBuilder) Summarize() string {
	s := "==== Pay2PubKeyHash ====\nSatNeeded:\t%d\nTxIns:\t1\nTxOuts:\t%d\n"
	return fmt.Sprintf(s, pkhB.SatNeeded(), pkhB.NumOuts)
}
