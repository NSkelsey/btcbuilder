package btcbuilder

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	_ "encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/conformal/btcec"
	"github.com/conformal/btcnet"
	"github.com/conformal/btcrpcclient"
	"github.com/conformal/btcscript"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwire"
	"github.com/jessevdk/go-flags"
)

var pver = btcwire.ProtocolVersion

// Network specific config
var magic btcwire.BitcoinNet

// Everything you need to spend from a txout in the UTXO
type TxInParams struct {
	TxOut    *btcwire.TxOut
	OutPoint *btcwire.OutPoint
	Wif      *btcutil.WIF
}

type BitcoinConf struct {
	RPCPassword string `long:"rpcpassword"`
	RPCUser     string `long:"rpcuser"`
	Testnet     bool   `long:"testnet" default:"false"`
	RPCListen   bool   `long:"server" default:"true"`
}

/*
	ConfigureApp assumes that you have a "bitcoin.conf" like ini file under the
	bitcoin data dir. If so it will build you an http rpc client and all the network
	parameters needed to configure your entire bitcoin based application. WARNING!
	This can and will die on you if it detects errors.
*/
func ConfigureApp() (*btcrpcclient.Client, btcnet.Params) {
	connCfg, testnet, err := CfgFromFile()
	if err != nil {
		log.Fatal(err)
	}

	client, err := makeRpcClient(connCfg)
	if err != nil {
		log.Fatal(err)
	}

	var params btcnet.Params
	if testnet {
		params = btcnet.TestNet3Params
	} else {
		params = btcnet.MainNetParams
	}

	return client, params
}

func CfgFromFile() (*btcrpcclient.ConnConfig, bool, error) {
	fileconf := &BitcoinConf{}

	path := btcutil.AppDataDir("bitcoin", false) + "/bitcoin.conf"

	parser := flags.NewParser(fileconf, flags.IgnoreUnknown)
	err := flags.NewIniParser(parser).ParseFile(path)
	if err != nil {
		return nil, false, err
	}

	if !fileconf.RPCListen {
		return nil, false, errors.New("Bitcoind not listening for rpc commands")
	}

	// TODO make rpcaddr configurable
	var rpcaddr string
	if fileconf.Testnet {
		rpcaddr = "127.0.0.1:18332"
	} else {
		rpcaddr = "127.0.0.1:8332"
	}

	// TODO use tls!
	connCfg := &btcrpcclient.ConnConfig{
		Host:         rpcaddr,
		User:         fileconf.RPCUser,
		Pass:         fileconf.RPCPassword,
		HttpPostMode: true,
		DisableTLS:   true,
	}

	return connCfg, fileconf.Testnet, nil
}

func NetParamsFromStr(name string) (*btcnet.Params, error) {
	var net btcnet.Params
	switch {
	case name == "TestNet3":
		net = btcnet.TestNet3Params
	case name == "MainNet":
		net = btcnet.MainNetParams
	case name == "SimNet":
		net = btcnet.SimNetParams
	case name == "TestNet":
		net = btcnet.RegressionNetParams
	default:
		return nil, errors.New(name + " is not a valid bitcoin network string")
	}
	return &net, nil
}

func makeRpcClient(connCfg *btcrpcclient.ConnConfig) (*btcrpcclient.Client, error) {
	client, err := btcrpcclient.New(connCfg, nil)
	if err != nil {
		return nil, err
	}
	err = checkconnection(client)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// check to see if we are connected
func checkconnection(client *btcrpcclient.Client) error {
	_, err := client.GetDifficulty()
	if err != nil {
		return err
	}
	return nil
}

func rpcTxPick(exact bool, targetAmnt int64, params BuilderParams) (*TxInParams, error) {
	// selects an unspent outpoint that is funded over the minAmount
	list, err := params.Client.ListUnspent()
	if err != nil {
		log.Println("list unpsent threw")
		return nil, err
	}
	if len(list) < 1 {
		return nil, errors.New("No unspent outputs at all.")
	}

	for _, prevJson := range list {
		_amnt, _ := btcutil.NewAmount(prevJson.Amount)
		amnt := int64(_amnt)
		txid := prevJson.TxId
		prevHash, _ := btcwire.NewShaHashFromStr(txid)
		outPoint := btcwire.NewOutPoint(prevHash, prevJson.Vout)

		_, contained := params.PendingSet[outPointStr(outPoint)]
		// This unpsent is in the pending set and it either exactly equals the target or
		// has a value above that target
		if !contained && (exact && targetAmnt == amnt || !exact && targetAmnt <= amnt) {
			// Found one, lets use it
			script, _ := hex.DecodeString(prevJson.ScriptPubKey)
			// None of the above ~should~ ever throw errors
			txOut := btcwire.NewTxOut(amnt, script)

			prevAddress, _ := btcutil.DecodeAddress(prevJson.Address, params.NetParams)
			wifkey, err := params.Client.DumpPrivKey(prevAddress)
			if err != nil {
				return nil, err
			}
			inParams := TxInParams{
				TxOut:    txOut,
				OutPoint: outPoint,
				Wif:      wifkey,
			}
			params.PendingSet[outPointStr(outPoint)] = struct{}{}
			return &inParams, nil
		}
	}
	// Never found a good outpoint
	return nil, errors.New("No txout with the right funds")
}

// specificUnspent gets an unspent output with an exact amount associated with it.
// it throws an error otherwise. It will also check to see if the tx selected is in the
// the pending tx set. If it is it will not use the txout
func specificUnspent(targetAmnt int64, params BuilderParams) (*TxInParams, error) {
	exact := true
	out, err := rpcTxPick(exact, targetAmnt, params)
	return out, err
}

// selectUnspent picks an unspent output that has atleast minAmount (sats) associated with it.
// Exactly similar to specific unspent except the operator is >=
func selectUnspent(minAmount int64, params BuilderParams) (*TxInParams, error) {
	exact := false
	out, err := rpcTxPick(exact, minAmount, params)
	return out, err
}

// composeUnspents Builds out a set of TxInParams that can be used to spend minAmount of bitcoin
func composeUnspents(minAmount int64, params BuilderParams) ([]*TxInParams, int64, error) {
	// Arbitrary constant!
	maxIns := 50

	totalIn := int64(0)
	inParamSet := make([]*TxInParams, 0)
	for i := 0; i < maxIns; i++ {
		txInParam, err := selectUnspent(minAmount/20, params)
		if err != nil {
			return nil, totalIn, err
		}
		inParamSet = append(inParamSet, txInParam)
		totalIn += txInParam.TxOut.Value
		if totalIn >= minAmount {
			return inParamSet, totalIn, nil
		}
	}
	msg := fmt.Sprintf("Do not have enough coins to compose input: %d, from %d", minAmount, totalIn)
	return inParamSet, 0, errors.New(msg)
}

// toHex converts a msgTx into a hex string.
func ToHex(tx *btcwire.MsgTx) string {
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	tx.Serialize(buf)
	txHex := hex.EncodeToString(buf.Bytes())
	return txHex
}

// generates a change output funding provided addr
func changeOutput(change, dustAmnt int64, addr btcutil.Address) (*btcwire.TxOut, bool) {
	if change < dustAmnt {
		return nil, false
	}
	script, _ := btcscript.PayToAddrScript(addr)
	txout := btcwire.NewTxOut(change, script)
	return txout, true
}

// sumOutputs derives the values in satoshis of tx.
func sumOutputs(tx *btcwire.MsgTx) (val int64) {
	val = 0
	for i := range tx.TxOut {
		val += tx.TxOut[i].Value
	}
	return val
}

func sumInputs(inParamSet []*TxInParams) (val int64) {
	val = 0
	for _, inpParam := range inParamSet {
		val += inpParam.TxOut.Value
	}
	return val
}

func newWifKeyPair(net *btcnet.Params) *btcutil.WIF {
	curve := elliptic.P256()
	priv, _ := ecdsa.GenerateKey(curve, rand.Reader)
	wif, _ := btcutil.NewWIF((*btcec.PrivateKey)(priv), net, true)
	return wif
}

func wifToAddr(wifkey *btcutil.WIF, net *btcnet.Params) btcutil.Address {
	pubkey := wifkey.SerializePubKey()
	pkHash := btcutil.Hash160(pubkey)
	addr, err := btcutil.NewAddressPubKeyHash(pkHash, net)
	if err != nil {
		log.Fatalf("failed to convert wif to address: %s\n", err)
	}
	return addr
}

// Gets a new address from an rpc client
func newAddr(client *btcrpcclient.Client) (btcutil.Address, error) {
	addr, err := client.GetNewAddress()
	if err != nil {
		return nil, err
	}
	return addr, nil
}

// prevOutVal looks up all the values of the oupoints used in the current tx
func PrevOutVal(tx *btcwire.MsgTx, client *btcrpcclient.Client) (int64, error) {
	// requires an rpc client and outpoints within wallets realm
	total := int64(0)
	for _, txin := range tx.TxIn {
		prevTxHash := txin.PreviousOutpoint.Hash
		var tx *btcutil.Tx
		tx, err := client.GetRawTransaction(&prevTxHash)
		if err != nil {
			return -1, err
		}
		vout := txin.PreviousOutpoint.Index
		txout := tx.MsgTx().TxOut[vout]
		total += txout.Value
	}
	return total, nil
}

func dataAddr(raw []byte, net *btcnet.Params) *btcutil.AddressPubKeyHash {
	addr, err := btcutil.NewAddressPubKeyHash(raw, net)
	if err != nil {
		log.Println(err)
	}
	return addr
}

func outPointStr(outpoint *btcwire.OutPoint) string {
	return fmt.Sprintf("%s[%d]", outpoint.Hash.String(), outpoint.Index)
}
