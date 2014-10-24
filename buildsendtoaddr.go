package btcbuilder

import (
	"fmt"

	"github.com/conformal/btcscript"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwire"
)

type ToAddrBuilder struct {
	Params BuilderParams
	Addr   btcutil.Address
}

func NewToAddrBuilder(params BuilderParams, addr string) *ToAddrBuilder {
	btcaddr, _ := btcutil.DecodeAddress(addr, params.NetParams)
	taB := ToAddrBuilder{
		Params: params,
		Addr:   btcaddr,
	}

	return &taB
}

func (builder *ToAddrBuilder) Build() (*btcwire.MsgTx, error) {

	utxo, err := selectUnspent(builder.SatNeeded(), builder.Params)
	if err != nil {
		return nil, err
	}

	txin := btcwire.NewTxIn(utxo.OutPoint, []byte{})

	msgtx := btcwire.NewMsgTx()
	msgtx.AddTxIn(txin)
	// add send to addr
	valout := builder.Params.InTarget - builder.Params.Fee
	outscript, _ := btcscript.PayToAddrScript(builder.Addr)
	txout := btcwire.NewTxOut(valout, outscript)

	msgtx.AddTxOut(txout)

	// add send to change addr
	total := utxo.TxOut.Value
	changeval := total - builder.SatNeeded()
	if changeval > builder.Params.DustAmnt {
		// Change needed
		changeAddr, err := builder.Params.Client.GetNewAddress()
		if err != nil {
			return nil, err
		}
		change, ok := changeOutput(changeval, builder.Params.DustAmnt, changeAddr)
		if ok {
			msgtx.AddTxOut(change)
		}
	}

	subscript := utxo.TxOut.PkScript
	privkey := utxo.Wif.PrivKey
	scriptSig, err := btcscript.SignatureScript(msgtx, 0, subscript, btcscript.SigHashAll, privkey, true)
	if err != nil {
		return nil, err
	}
	txin.SignatureScript = scriptSig

	return msgtx, nil
}

func (taB *ToAddrBuilder) SatNeeded() int64 {
	return taB.Params.InTarget
}

func (taB *ToAddrBuilder) Summarize() string {
	s := "==== Send To Addr ====\nSatNeeded:\t%d\nTxIns:\t?\nTxOuts:\t2"
	return fmt.Sprintf(s, taB.SatNeeded())
}

func (taB *ToAddrBuilder) Log(msg string) {
	taB.Params.Logger.Println(msg)
}
