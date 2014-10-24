package btcbuilder

import (
	"fmt"
	"math"

	"github.com/conformal/btcscript"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwire"
)

// A Standard multi sig builder using the bip11 method
// IE OP_0 m [pub keys] n OP_CHECKMULTISIG
type MultiSigBuilder struct {
	Params     BuilderParams
	M          int64      // min sigs needed for every tx out
	N          int64      // number of sigs
	PubKeyList [][][]byte // the list of raw pubkeys to insert into txouts
}

func NewMultiSigBuilder(params BuilderParams, m int64, pklist [][][]byte) *MultiSigBuilder {
	numSigs := 3
	msb := MultiSigBuilder{
		Params:     params,
		M:          m,
		N:          int64(numSigs),
		PubKeyList: pklist,
	}
	return &msb
}

func (msB *MultiSigBuilder) SatNeeded() int64 {
	sum := msB.Params.InTarget
	return sum
}

func (msB *MultiSigBuilder) eachOutVal() int64 {
	numouts := int64(len(msB.PubKeyList))
	total := msB.Params.InTarget
	fee := msB.Params.Fee

	return (total - fee) / numouts
}

// TODO This will add multisig Txouts to the unspent set be AWARE
func (msB *MultiSigBuilder) Build() (*btcwire.MsgTx, error) {

	utxo, err := specificUnspent(msB.SatNeeded(), msB.Params)
	if err != nil {
		return nil, err
	}
	msgtx := btcwire.NewMsgTx()

	txin := btcwire.NewTxIn(utxo.OutPoint, []byte{})
	msgtx.AddTxIn(txin)

	for _, pubkeys := range msB.PubKeyList {
		// M pubkey pubkey pubkey N OP_CHECKMULTISIG
		scriptBuilder := btcscript.NewScriptBuilder().AddInt64(msB.M)
		for _, pk := range pubkeys {
			scriptBuilder = scriptBuilder.AddData(pk)
		}
		scriptBuilder = scriptBuilder.AddInt64(msB.N).AddOp(btcscript.OP_CHECKMULTISIG)
		PkScript := scriptBuilder.Script()
		txout := btcwire.NewTxOut(msB.eachOutVal(), PkScript)
		msgtx.AddTxOut(txout)
	}

	// Sign this puppy
	privkey := utxo.Wif.PrivKey
	subscript := utxo.TxOut.PkScript
	sigflag := btcscript.SigHashAll
	scriptSig, err := btcscript.SignatureScript(msgtx, 0, subscript,
		sigflag, privkey, true)
	if err != nil {
		return nil, err
	}

	msgtx.TxIn[0].SignatureScript = scriptSig

	return msgtx, nil
}

func (msB *MultiSigBuilder) Log(msg string) {
	if msB.Params.Logger != nil {
		msB.Params.Logger.Println(msg)
	}
}

func (msB *MultiSigBuilder) Summarize() string {
	s := "==== MuliSig ====\nSatNeeded:\t%d\nTxIns:\t1\nTxOuts:\t%d\n"
	return fmt.Sprintf(s, msB.SatNeeded(), len(msB.PubKeyList))
}

func CreateList(data []byte, keys ...*btcutil.WIF) [][][]byte {
	dataOuts := int(math.Ceil(float64(len(data)) / 65))
	numTxOuts := (dataOuts + len(keys)) / 3
	outMatrix := make([][][]byte, numTxOuts)

	for i := 0; i < numTxOuts; i++ {
		txOutData := make([][]byte, 3)

		// copy keys in
		for j, key := range keys {
			txOutData[j] = key.SerializePubKey()
		}

		for k := len(keys); k < 3; k++ {
			// copy data into dest
			dest := make([]byte, 65, 65)
			if len(data) > 0 {
				m := copy(dest, data)
				data = data[m:]
			}
			txOutData[k] = dest
		}
		outMatrix[i] = txOutData
	}
	return outMatrix
}
