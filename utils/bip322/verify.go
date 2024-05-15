package bip322

import (
	"crypto/sha256"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func GetSha256(data []byte) (hash []byte) {
	sha := sha256.New()
	sha.Write(data[:])
	hash = sha.Sum(nil)
	return
}

func GetTagSha256(data []byte) (hash []byte) {
	tag := []byte("BIP0322-signed-message")
	hashTag := GetSha256(tag)
	var msg []byte
	msg = append(msg, hashTag...)
	msg = append(msg, hashTag...)
	msg = append(msg, data...)
	return GetSha256(msg)
}

func PrepareTx(pkScript []byte, message string) (toSign *wire.MsgTx, err error) {
	// Create a new transaction to spend
	toSpend := wire.NewMsgTx(0)

	// Decode the message hash
	messageHash := GetTagSha256([]byte(message))

	// Create the script for to_spend
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_0)
	builder.AddData(messageHash)
	scriptSig, err := builder.Script()
	if err != nil {
		return nil, err
	}

	// Create a TxIn with the outpoint 000...000:FFFFFFFF
	prevOutHash, _ := chainhash.NewHashFromStr("0000000000000000000000000000000000000000000000000000000000000000")
	prevOut := wire.NewOutPoint(prevOutHash, wire.MaxPrevOutIndex)
	txIn := wire.NewTxIn(prevOut, scriptSig, nil)
	txIn.Sequence = 0

	toSpend.AddTxIn(txIn)
	toSpend.AddTxOut(wire.NewTxOut(0, pkScript))

	// Create a transaction for to_sign
	toSign = wire.NewMsgTx(0)
	hash := toSpend.TxHash()

	prevOutSpend := wire.NewOutPoint((*chainhash.Hash)(hash.CloneBytes()), 0)

	txSignIn := wire.NewTxIn(prevOutSpend, nil, nil)
	txSignIn.Sequence = 0
	toSign.AddTxIn(txSignIn)

	// Create the script for to_sign
	builderPk := txscript.NewScriptBuilder()
	builderPk.AddOp(txscript.OP_RETURN)
	scriptPk, err := builderPk.Script()
	if err != nil {
		return nil, err
	}
	toSign.AddTxOut(wire.NewTxOut(0, scriptPk))
	return toSign, nil
}

// VerifySignature
// signature: 64B, pkScript: 33B, message: any
func VerifySignature(witness wire.TxWitness, pkScript []byte, message string) bool {
	toSign, err := PrepareTx(pkScript, message)
	if err != nil {
		fmt.Println("verifying signature, PrepareTx failed:", err)
		return false
	}

	toSign.TxIn[0].Witness = witness
	prevFetcher := txscript.NewCannedPrevOutputFetcher(
		pkScript, 0,
	)
	hashCache := txscript.NewTxSigHashes(toSign, prevFetcher)
	vm, err := txscript.NewEngine(pkScript, toSign, 0, txscript.StandardVerifyFlags, nil, hashCache, 0, prevFetcher)
	if err != nil {
		return false
	}
	if err := vm.Execute(); err != nil {
		return false
	}
	return true
}

func SignSignatureTaproot(pkey, message string) (witness wire.TxWitness, pkScript []byte, err error) {
	decodedWif, err := btcutil.DecodeWIF(pkey)
	if err != nil {
		return nil, nil, err
	}

	privKey := decodedWif.PrivKey
	pubKey := txscript.ComputeTaprootKeyNoScript(privKey.PubKey())
	pkScript, err = utils.PayToTaprootScript(pubKey)

	toSign, err := PrepareTx(pkScript, message)
	if err != nil {
		return nil, nil, err
	}

	prevFetcher := txscript.NewCannedPrevOutputFetcher(
		pkScript, 0,
	)
	sigHashes := txscript.NewTxSigHashes(toSign, prevFetcher)

	witness, err = txscript.TaprootWitnessSignature(
		toSign, sigHashes, 0, 0, pkScript,
		txscript.SigHashDefault, privKey,
	)
	return witness, pkScript, nil
}

func SignSignatureP2WPKH(pkey, message string) (witness wire.TxWitness, pkScript []byte, err error) {
	decodedWif, err := btcutil.DecodeWIF(pkey)
	if err != nil {
		return nil, nil, err
	}

	privKey := decodedWif.PrivKey
	pubKey := privKey.PubKey()
	pkScript, err = utils.PayToWitnessScript(pubKey)

	toSign, err := PrepareTx(pkScript, message)
	if err != nil {
		return nil, nil, err
	}

	prevFetcher := txscript.NewCannedPrevOutputFetcher(
		pkScript, 0,
	)
	sigHashes := txscript.NewTxSigHashes(toSign, prevFetcher)

	witness, err = txscript.WitnessSignature(toSign, sigHashes,
		0, 0, pkScript, txscript.SigHashAll,
		privKey, true)
	if err != nil {
		return nil, nil, err
	}
	return witness, pkScript, nil
}
