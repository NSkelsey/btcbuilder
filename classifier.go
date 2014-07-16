package btcbuilder

import (
	"sort"

	"github.com/conformal/btcscript"
	"github.com/conformal/btcwire"
)

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
	// If the tx does not have funky output scripts just count occurrences
	sort.Sort(pl)
	return pl[0].Class.String()
}
