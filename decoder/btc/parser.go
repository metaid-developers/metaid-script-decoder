package btc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/metaid-developers/metaid-script-decoder/decoder"
	"github.com/metaid-developers/metaid-script-decoder/decoder/common"
)

// BTCParser is the BTC chain parser
type BTCParser struct {
	config *decoder.ParserConfig
}

// NewBTCParser creates a BTC parser
func NewBTCParser(config *decoder.ParserConfig) *BTCParser {
	if config == nil {
		config = decoder.DefaultConfig()
	}
	return &BTCParser{
		config: config,
	}
}

// GetChainName returns the chain name
func (p *BTCParser) GetChainName() string {
	return "btc"
}

// ParseTransaction parses a BTC transaction
func (p *BTCParser) ParseTransaction(txBytes []byte, chainParams interface{}) ([]*decoder.Pin, error) {
	// Parse chainParams
	params, ok := chainParams.(*chaincfg.Params)
	if !ok && chainParams != nil {
		return nil, fmt.Errorf("invalid chainParams type for BTC, expected *chaincfg.Params")
	}
	if params == nil {
		params = &chaincfg.MainNetParams
	}

	// Deserialize transaction
	msgTx := wire.NewMsgTx(wire.TxVersion)
	if err := msgTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %w", err)
	}

	var pins []*decoder.Pin

	// 1. First check for OP_RETURN format PINs
	opReturnPins := p.parseOpReturnPins(msgTx, params)
	if len(opReturnPins) > 0 {
		pins = append(pins, opReturnPins...)
		return pins, nil
	}

	// 2. Check for Witness format PINs
	witnessPins := p.parseWitnessPins(msgTx, params)
	pins = append(pins, witnessPins...)

	return pins, nil
}

// parseOpReturnPins parses OP_RETURN format PINs
func (p *BTCParser) parseOpReturnPins(msgTx *wire.MsgTx, params *chaincfg.Params) []*decoder.Pin {
	var pins []*decoder.Pin
	txHash := msgTx.TxHash().String()

	for i, out := range msgTx.TxOut {
		class, _, _, _ := txscript.ExtractPkScriptAddrs(out.PkScript, params)
		if class.String() == "nonstandard" {
			pin := p.parseOpReturnScript(out.PkScript)
			if pin == nil {
				continue
			}

			// Get PIN owner address
			address, vout := p.getOpReturnOwner(msgTx, params)
			if address == "" {
				continue
			}

			pin.TxID = txHash
			pin.Vout = uint32(vout)
			pin.OwnerAddress = address
			pin.OwnerMetaId = common.CalculateMetaId(address)
			pin.ChainName = "btc"
			pin.InscriptionTxIndex = i

			pins = append(pins, pin)
			break // Usually only one OP_RETURN
		}
	}

	return pins
}

// parseWitnessPins parses Witness format PINs
func (p *BTCParser) parseWitnessPins(msgTx *wire.MsgTx, params *chaincfg.Params) []*decoder.Pin {
	var pins []*decoder.Pin
	txHash := msgTx.TxHash().String()

	for i, txIn := range msgTx.TxIn {
		// Check witness data
		if len(txIn.Witness) <= 1 {
			continue
		}
		if len(txIn.Witness[len(txIn.Witness)-1]) <= 1 {
			continue
		}

		// Taproot Annex check
		if len(txIn.Witness) == 2 && txIn.Witness[len(txIn.Witness)-1][0] == txscript.TaprootAnnexTag {
			continue
		}

		// Get witness script
		var witnessScript []byte
		if txIn.Witness[len(txIn.Witness)-1][0] == txscript.TaprootAnnexTag {
			witnessScript = txIn.Witness[len(txIn.Witness)-1]
		} else {
			if len(txIn.Witness) >= 2 {
				witnessScript = txIn.Witness[len(txIn.Witness)-2]
			}
		}

		if len(witnessScript) == 0 {
			continue
		}

		// Parse PIN
		pin := p.parseWitnessScript(witnessScript)
		if pin == nil {
			continue
		}

		// Get PIN owner address
		address, vout, outValue, locationIdx := p.getWitnessOwner(msgTx, i, params)
		if address == "" {
			address = "unknown"
			vout = 0
		}

		pin.Id = fmt.Sprintf("%si%d", txHash, vout)
		pin.TxID = txHash
		pin.Vout = uint32(vout)
		pin.OwnerAddress = address
		pin.OwnerMetaId = common.CalculateMetaId(address)
		pin.ChainName = "btc"
		pin.InscriptionTxIndex = i

		//// PIN location
		pin.Location = fmt.Sprintf("%s:%d:%d", txHash, vout, locationIdx)
		pin.Offset = uint64(vout)
		pin.Output = fmt.Sprintf("%s:%d", txHash, vout)
		pin.OutputValue = outValue

		pins = append(pins, pin)
	}

	return pins
}

// parseOpReturnScript parses OP_RETURN scripts
func (p *BTCParser) parseOpReturnScript(pkScript []byte) *decoder.Pin {
	tokenizer := txscript.MakeScriptTokenizer(0, pkScript)
	for tokenizer.Next() {
		if tokenizer.Opcode() == txscript.OP_RETURN {
			if !tokenizer.Next() || hex.EncodeToString(tokenizer.Data()) != p.config.ProtocolID {
				return nil
			}
			return p.parseOnePin(&tokenizer)
		}
	}
	return nil
}

// parseWitnessScript parses Witness scripts
func (p *BTCParser) parseWitnessScript(witnessScript []byte) *decoder.Pin {
	tokenizer := txscript.MakeScriptTokenizer(0, witnessScript)
	for tokenizer.Next() {
		// Check inscription envelope header: OP_FALSE(0x00), OP_IF(0x63), PROTOCOL_ID
		if tokenizer.Opcode() == txscript.OP_FALSE {
			if !tokenizer.Next() || tokenizer.Opcode() != txscript.OP_IF {
				return nil
			}
			if !tokenizer.Next() || hex.EncodeToString(tokenizer.Data()) != p.config.ProtocolID {
				return nil
			}
			return p.parseOnePin(&tokenizer)
		}
	}
	return nil
}

// parseOnePin parses a single PIN data
func (p *BTCParser) parseOnePin(tokenizer *txscript.ScriptTokenizer) *decoder.Pin {
	var infoList [][]byte

	// Collect all data
	for tokenizer.Next() {
		if tokenizer.Opcode() == txscript.OP_ENDIF {
			break
		}
		infoList = append(infoList, tokenizer.Data())
		if len(tokenizer.Data()) > 520 {
			return nil
		}
	}

	// Check for errors
	if err := tokenizer.Err(); err != nil {
		return nil
	}

	if len(infoList) < 1 {
		return nil
	}

	pin := &decoder.Pin{}
	pin.Operation = strings.ToLower(string(infoList[0]))

	// revoke operation requires at least 5 fields
	if pin.Operation == "revoke" && len(infoList) < 5 {
		return nil
	}

	// Other operations require at least 6 fields
	if len(infoList) < 6 && pin.Operation != "revoke" {
		return nil
	}

	// Parse each field
	pin.Path = common.NormalizePath(string(infoList[1]))
	pin.ParentPath = common.GetParentPath(pin.Path)

	encryption := "0"
	if len(infoList) > 2 && infoList[2] != nil {
		encryption = string(infoList[2])
	}
	pin.Encryption = encryption

	version := "0"
	if len(infoList) > 3 && infoList[3] != nil {
		version = string(infoList[3])
	}
	pin.Version = version

	contentType := "application/json"
	if len(infoList) > 4 && infoList[4] != nil {
		contentType = common.NormalizeContentType(string(infoList[4]))
	}
	pin.ContentType = contentType

	// Merge remaining body data
	var body []byte
	for i := 5; i < len(infoList); i++ {
		body = append(body, infoList[i]...)
	}
	pin.ContentBody = body
	pin.ContentLength = uint64(len(body))

	return pin
}

// getOpReturnOwner gets the owner of an OP_RETURN format PIN
func (p *BTCParser) getOpReturnOwner(tx *wire.MsgTx, params *chaincfg.Params) (address string, vout int) {
	for i, out := range tx.TxOut {
		class, addresses, _, _ := txscript.ExtractPkScriptAddrs(out.PkScript, params)
		if class.String() != "nonstandard" && len(addresses) > 0 {
			vout = i
			address = addresses[0].EncodeAddress()
			return
		}
	}
	return "", 0
}

// getWitnessOwner gets the owner of a Witness format PIN
func (p *BTCParser) getWitnessOwner(tx *wire.MsgTx, inIdx int, params *chaincfg.Params) (address string, vout int, outValue int64, locationIdx int64) {
	// Simple case: single input or single output
	if len(tx.TxIn) == 1 || len(tx.TxOut) == 1 || inIdx == 0 {
		if len(tx.TxOut) > 0 {
			_, addresses, _, _ := txscript.ExtractPkScriptAddrs(tx.TxOut[0].PkScript, params)
			if len(addresses) > 0 {
				address = addresses[0].EncodeAddress()
				vout = 0
				outValue = tx.TxOut[0].Value
				locationIdx = 0
			}
		}
		return
	}

	// For multiple inputs/outputs, return the first output
	// Note: Complete owner determination requires querying input transactions, which needs an external node service
	// Here we simplify by only returning the first valid output
	if len(tx.TxOut) > 0 {
		_, addresses, _, _ := txscript.ExtractPkScriptAddrs(tx.TxOut[0].PkScript, params)
		if len(addresses) > 0 {
			address = addresses[0].EncodeAddress()
			vout = 0
		}
	}

	return
}
