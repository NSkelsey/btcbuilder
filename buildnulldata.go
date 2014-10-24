package btcbuilder

import (
	"errors"
	"fmt"

	"github.com/conformal/btcscript"
	"github.com/conformal/btcwire"
)

type NullDataBuilder struct {
	Params BuilderParams
	Data   []byte
	Change bool
}

func NewNullData(params BuilderParams, data []byte, change bool) *NullDataBuilder {
	ndB := NullDataBuilder{
		Params: params,
		Data:   data,
		Change: change,
	}
	return &ndB
}

func (ndB *NullDataBuilder) SatNeeded() (sum int64) {
	sum = 0
	if ndB.Change {
		sum = ndB.Params.InTarget
	} else {
		sum = ndB.Params.DustAmnt + ndB.Params.Fee
	}
	return sum
}

func (ndB *NullDataBuilder) Build() (*btcwire.MsgTx, error) {

	utxo, err := specificUnspent(ndB.SatNeeded(), ndB.Params)
	if err != nil {
		return nil, err
	}

	msgtx := btcwire.NewMsgTx()

	if len(ndB.Data) > 40 {
		return nil, errors.New("Data is too long to make this a standard tx.")
	}

	// OP Return output
	retbuilder := btcscript.NewScriptBuilder().AddOp(btcscript.OP_RETURN).AddData(ndB.Data)
	op_return := btcwire.NewTxOut(0, retbuilder.Script())
	msgtx.AddTxOut(op_return)

	if ndB.Change {
		// change ouput
		addr, _ := newAddr(ndB.Params.Client)
		change, ok := changeOutput(ndB.SatNeeded()-ndB.Params.Fee, ndB.Params.DustAmnt, addr)
		if !ok {
			return nil, errors.New("Not enough for change")
		}
		msgtx.AddTxOut(change)
	}

	// funding input
	txin := btcwire.NewTxIn(utxo.OutPoint, []byte{})
	msgtx.AddTxIn(txin)

	// sign msgtx
	privkey := utxo.Wif.PrivKey
	scriptSig, err := btcscript.SignatureScript(msgtx, 0, utxo.TxOut.PkScript, btcscript.SigHashAll, privkey, true)
	if err != nil {
		return nil, err
	}
	txin.SignatureScript = scriptSig

	return msgtx, nil
}

func (ndB *NullDataBuilder) Log(msg string) {
	ndB.Params.Logger.Println(msg)
}

func (ndB *NullDataBuilder) Summarize() string {
	s := "==== NullData ====\nSatNeeded:\t%d\nTxIns:\t1\nTxOuts:\t%d\nLenData:\t%d\n"
	numouts := 1
	if ndB.Change {
		numouts = 2
	}
	return fmt.Sprintf(s, ndB.SatNeeded(), numouts, len(ndB.Data))
}
