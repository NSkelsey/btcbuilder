package btcbuilder

import (
	"fmt"

	"github.com/NSkelsey/protocol/ahimsa"
	"github.com/conformal/btcscript"
	"github.com/conformal/btcwire"
)

type BulletinBuilder struct {
	Bulletin ahimsa.Bulletin
	BurnAmnt int64
	Params   BuilderParams
}

func NewBulletinBuilder(params BuilderParams, burnAmnt int64, bltn ahimsa.Bulletin) *BulletinBuilder {
	// Initializes a bulletin builder which will be ready to build the Bulletin
	// provided. BurnAmnt referenes the amount of satoshi to put behind a single
	// txout which will be used to push data. Note that this builder will generate
	// change.

	bltnB := &BulletinBuilder{
		Bulletin: bltn,
		Params:   params,
		BurnAmnt: burnAmnt,
	}
	return bltnB
}

func (bltnB *BulletinBuilder) SatNeeded() int64 {
	// Returns the satoshi needed to generate this bulletin

	numouts, _ := bltnB.Bulletin.NumOuts()
	msgcost := int64(numouts) * bltnB.BurnAmnt

	totalcost := msgcost + bltnB.Params.Fee
	return totalcost
}

func (bltnB *BulletinBuilder) Build() (*btcwire.MsgTx, error) {
	utxo, err := selectUnspent(bltnB.SatNeeded(), bltnB.Params)
	if err != nil {
		return nil, err
	}
	msgtx := btcwire.NewMsgTx()
	// Add data storing txouts.
	txouts, err := bltnB.Bulletin.TxOuts(bltnB.BurnAmnt, bltnB.Params.NetParams)
	if err != nil {
		return nil, err
	}
	msgtx.TxOut = txouts

	txin := btcwire.NewTxIn(utxo.OutPoint, []byte{})
	msgtx.AddTxIn(txin)

	// Deal with change
	changeAmnt := utxo.TxOut.Value - bltnB.SatNeeded()
	if changeAmnt > bltnB.Params.DustAmnt {
		changeOut, err := makeChange(changeAmnt, bltnB.Params)
		if err != nil {
			return nil, err
		}
		msgtx.AddTxOut(changeOut)
	}

	// Sign the Bulletin
	privkey := utxo.Wif.PrivKey.ToECDSA()
	scriptSig, err := btcscript.SignatureScript(msgtx, 0, utxo.TxOut.PkScript, btcscript.SigHashAll, privkey, true)
	if err != nil {
		return nil, err
	}
	txin.SignatureScript = scriptSig
	return msgtx, nil
}

func (bltnB *BulletinBuilder) SetBulletin(bltn *ahimsa.Bulletin) {
	bltnB.Bulletin = *bltn
}

func (b *BulletinBuilder) Log(msg string) {
	b.Params.Logger.Println(msg)
}

func (bltnB *BulletinBuilder) Summarize() string {
	s := "==== Bulletin ====\nSatNeeded:\t%d\nTxIns:\t1\nTxOuts:\t%d\nLenData:\t%d\n"
	numouts, _ := bltnB.Bulletin.NumOuts()
	rawB, _ := bltnB.Bulletin.Bytes()
	return fmt.Sprintf(s, bltnB.SatNeeded(), numouts, len(rawB))
}
