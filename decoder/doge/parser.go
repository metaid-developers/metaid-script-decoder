package doge

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/metaid-developers/metaid-script-decoder/decoder"
	"github.com/metaid-developers/metaid-script-decoder/decoder/common"
)

// DOGEParser is the DOGE chain parser
type DOGEParser struct {
	config *decoder.ParserConfig
}

// NewDOGEParser creates a DOGE parser
func NewDOGEParser(config *decoder.ParserConfig) *DOGEParser {
	if config == nil {
		config = decoder.DefaultConfig()
	}
	return &DOGEParser{
		config: config,
	}
}

// GetChainName returns the chain name
func (p *DOGEParser) GetChainName() string {
	return "doge"
}

// ParseTransaction parses a DOGE transaction
func (p *DOGEParser) ParseTransaction(txBytes []byte, chainParams interface{}) ([]*decoder.Pin, error) {
	// Parse chainParams
	params, ok := chainParams.(*chaincfg.Params)
	if !ok && chainParams != nil {
		return nil, fmt.Errorf("invalid chainParams type for DOGE, expected *chaincfg.Params")
	}
	if params == nil {
		params = &DogeMainNetParams
	}

	// Deserialize transaction
	msgTx := wire.NewMsgTx(wire.TxVersion)
	if err := msgTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %w", err)
	}

	var pins []*decoder.Pin

	// DOGE uses ScriptSig format (P2SH redeem script), not Witness
	scriptSigPins := p.parseScriptSigPins(msgTx, params)
	pins = append(pins, scriptSigPins...)

	return pins, nil
}

// parseScriptSigPins parses ScriptSig format PINs
func (p *DOGEParser) parseScriptSigPins(msgTx *wire.MsgTx, params *chaincfg.Params) []*decoder.Pin {
	var pins []*decoder.Pin
	txHash := msgTx.TxHash().String()

	// Dogecoin: Parse inscriptions from ScriptSig (P2SH redeem script)
	// Unlike Bitcoin's SegWit which uses witness data, Dogecoin uses legacy P2SH
	for i, input := range msgTx.TxIn {
		// Check if ScriptSig exists
		if len(input.SignatureScript) == 0 {
			continue
		}

		// Try parsing direct format first (metaid data at the beginning of ScriptSig)
		pin := p.parsePinFromDirectScriptSig(input.SignatureScript)
		if pin == nil {
			// Parse ScriptSig to extract the redeem script
			// ScriptSig format for P2SH: <signature> <redeemScript>
			tokenizer := txscript.MakeScriptTokenizer(0, input.SignatureScript)
			var redeemScript []byte
			var lastData []byte

			// Iterate through ScriptSig to find the redeem script (last push data)
			for tokenizer.Next() {
				if len(tokenizer.Data()) > 0 {
					lastData = tokenizer.Data()
				}
			}

			// The last pushed data in ScriptSig should be the redeem script
			if len(lastData) > 0 {
				redeemScript = lastData
			} else {
				continue
			}

			// Parse the redeem script for inscription data
			pin = p.parsePinFromRedeemScript(redeemScript)
		}

		if pin == nil {
			continue
		}

		// Get PIN owner address
		address, vout, outValue, locationIdx := p.getScriptSigOwner(msgTx, i, params)
		if address == "" {
			address = ""
			vout = 0
		}

		pin.Id = fmt.Sprintf("%si%d", txHash, vout)
		pin.TxID = txHash
		pin.Vout = uint32(vout)
		pin.OwnerAddress = address
		pin.OwnerMetaId = common.CalculateMetaId(address)
		pin.ChainName = "doge"
		pin.InscriptionTxIndex = i
		pin.CreatorInputTxVinLocation = fmt.Sprintf("%s:%d", input.PreviousOutPoint.Hash.String(), input.PreviousOutPoint.Index)

		// PIN location
		pin.Location = fmt.Sprintf("%s:%d:%d", txHash, vout, locationIdx)
		pin.Offset = uint64(vout)
		pin.Output = fmt.Sprintf("%s:%d", txHash, vout)
		pin.OutputValue = outValue

		pins = append(pins, pin)
	}

	return pins
}

// parsePinFromRedeemScript parses Dogecoin inscription data from P2SH redeem script
// Dogecoin inscription format in redeem script:
// <pubkey> OP_CHECKSIGVERIFY OP_FALSE OP_IF "metaid" <operation> <path> <encryption> <version> <contentType> <content> [more content...] OP_ENDIF
func (p *DOGEParser) parsePinFromRedeemScript(redeemScript []byte) *decoder.Pin {
	tokenizer := txscript.MakeScriptTokenizer(0, redeemScript)

	// Skip the pubkey and OP_CHECKSIGVERIFY at the beginning
	if !tokenizer.Next() {
		return nil
	}
	if !tokenizer.Next() || tokenizer.Opcode() != txscript.OP_CHECKSIGVERIFY {
		return nil
	}

	// Look for inscription envelope: OP_FALSE OP_IF
	if !tokenizer.Next() || tokenizer.Opcode() != txscript.OP_FALSE {
		return nil
	}
	if !tokenizer.Next() || tokenizer.Opcode() != txscript.OP_IF {
		return nil
	}

	// Check for protocol ID marker
	// Can be either string "metaid" or hex encoded protocol ID
	if !tokenizer.Next() {
		return nil
	}
	protocolMarker := string(tokenizer.Data())
	protocolIDHex := hex.EncodeToString(tokenizer.Data())
	// Check both string format and hex format
	if protocolMarker != "metaid" && protocolIDHex != p.config.ProtocolID {
		return nil
	}

	// Parse inscription data following the standard metaid protocol format
	// Format: protocolID <operation> <path> <encryption> <version> <contentType> <content> [more content...]
	// Collect all data fields until OP_ENDIF
	var infoList [][]byte
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

	return p.parseOnePin(infoList)
}

// parsePinFromDirectScriptSig parses Dogecoin inscription data directly from ScriptSig
// This format has metaid protocol data at the beginning of ScriptSig without OP_IF/OP_ENDIF wrapper
// Format: <pushdata protocolID> <pushdata operation> <pushdata contentType> <pushdata encryption> <pushdata version> <pushdata address:path> <pushdata content> <signature> <pubkey> ...
func (p *DOGEParser) parsePinFromDirectScriptSig(scriptSig []byte) *decoder.Pin {
	if len(scriptSig) < 7 {
		return nil
	}

	tokenizer := txscript.MakeScriptTokenizer(0, scriptSig)
	var infoList [][]byte

	// Collect all push data from ScriptSig
	for tokenizer.Next() {
		if len(tokenizer.Data()) > 0 {
			infoList = append(infoList, tokenizer.Data())
		}
	}

	if err := tokenizer.Err(); err != nil {
		return nil
	}

	if len(infoList) < 6 {
		return nil
	}

	// Check if first field is "metaid" protocol marker (as string, not hex)
	protocolMarker := string(infoList[0])
	// Convert to hex for comparison with config
	protocolIDHex := hex.EncodeToString(infoList[0])
	if protocolMarker != "metaid" && protocolIDHex != p.config.ProtocolID {
		return nil
	}

	pin := &decoder.Pin{}
	pin.Operation = strings.ToLower(string(infoList[1]))

	// // Special case: init operation
	// if pin.Operation == "init" {
	// 	pin.Path = "/"
	// 	pin.ParentPath = ""
	// 	pin.Encryption = "0"
	// 	pin.Version = "0"
	// 	pin.ContentType = "application/json"
	// 	return pin
	// }

	// Validate operation
	if pin.Operation != "create" && pin.Operation != "modify" && pin.Operation != "revoke" {
		return nil
	}

	// For revoke, we need at least 6 fields; for others, at least 7
	if pin.Operation == "revoke" && len(infoList) < 6 {
		return nil
	}
	if pin.Operation != "revoke" && len(infoList) < 7 {
		return nil
	}

	// Parse content type (field 2)
	contentType := "application/json"
	if len(infoList) > 2 && len(infoList[2]) > 0 {
		contentType = common.NormalizeContentType(string(infoList[2]))
	}
	pin.ContentType = contentType

	// Parse encryption (field 3)
	encryption := "0"
	if len(infoList) > 3 && len(infoList[3]) > 0 {
		encryption = string(infoList[3])
	}
	pin.Encryption = encryption

	// Parse version (field 4)
	version := "0"
	if len(infoList) > 4 && len(infoList[4]) > 0 {
		version = string(infoList[4])
	}
	pin.Version = version

	// Parse field 5: address:path format
	// Example: "bc1p20k3x2c4mglfxr5wa5sgtgechwstpld80kru2cg4gmm4urvuaqqsvapxu0:/protocols/simplegroupchat"
	if len(infoList) > 5 && len(infoList[5]) > 0 {
		field5 := string(infoList[5])
		// Split by colon to separate address and path
		parts := strings.SplitN(field5, ":", 2)
		if len(parts) == 2 {
			// Use the path part (after colon)
			pin.Path = common.NormalizePath(parts[1])
		} else {
			// If no colon found, treat entire field as path
			pin.Path = common.NormalizePath(field5)
		}
	} else {
		// Default path if field 5 is missing
		pin.Path = "/info"
	}
	pin.ParentPath = common.GetParentPath(pin.Path)

	// Parse content body (field 6 onwards)
	// Stop if this looks like a signature (starts with 0x30 and is 70-73 bytes)
	// or a pubkey (33 or 65 bytes starting with 0x02, 0x03, 0x04)
	var body []byte
	for i := 6; i < len(infoList); i++ {
		data := infoList[i]
		// Stop if this looks like a signature or pubkey
		if len(data) >= 70 && len(data) <= 73 && data[0] == 0x30 {
			break
		}
		if (len(data) == 33 || len(data) == 65) && (data[0] == 0x02 || data[0] == 0x03 || data[0] == 0x04) {
			break
		}
		body = append(body, data...)
	}

	pin.ContentBody = body
	pin.ContentLength = uint64(len(body))

	return pin
}

// parseOnePin parses a single PIN data
func (p *DOGEParser) parseOnePin(infoList [][]byte) *decoder.Pin {
	if len(infoList) < 1 {
		return nil
	}

	pin := &decoder.Pin{}
	pin.Operation = strings.ToLower(string(infoList[0]))

	// Special case: init operation
	if pin.Operation == "init" {
		pin.Path = "/"
		pin.ParentPath = ""
		pin.Encryption = "0"
		pin.Version = "0"
		pin.ContentType = "application/json"
		return pin
	}

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

// getScriptSigOwner gets the owner of a ScriptSig format PIN
func (p *DOGEParser) getScriptSigOwner(tx *wire.MsgTx, inIdx int, params *chaincfg.Params) (address string, vout int, outValue int64, locationIdx int64) {
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
			outValue = tx.TxOut[0].Value
			locationIdx = 0
		}
	}

	return
}

// DogeMainNetParams defines the network parameters for the main Dogecoin network.
var DogeMainNetParams = chaincfg.Params{
	Name:        "mainnet",
	Net:         wire.BitcoinNet(0xc0c0c0c0), // Dogecoin MainNet magic
	DefaultPort: "22556",
	DNSSeeds: []chaincfg.DNSSeed{
		{Host: "seed.dogecoin.com", HasFiltering: true},
		{Host: "seed.multidoge.org", HasFiltering: true},
	},
	GenesisHash:      newHashFromStr("1a91e3dace36e2be3bf030a65679fe821aa1d6ef92e7c9902eb318182c355691"),
	PowLimit:         newBigIntFromHex("00000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
	CoinbaseMaturity: 240,
	PubKeyHashAddrID: 0x1e,   // starts with D
	ScriptHashAddrID: 0x16,   // starts with 9
	PrivateKeyID:     0x9e,   // starts with Q
	Bech32HRPSegwit:  "doge", // Not widely used but defined
	HDCoinType:       3,
}

// DogeTestNetParams defines the network parameters for the test Dogecoin network.
var DogeTestNetParams = chaincfg.Params{
	Name:        "testnet",
	Net:         wire.BitcoinNet(0xfcc1b7dc), // Dogecoin TestNet magic
	DefaultPort: "44556",
	DNSSeeds: []chaincfg.DNSSeed{
		{Host: "testseed.jrn.me.uk", HasFiltering: true},
	},
	GenesisHash:      newHashFromStr("bb0a78264637406b6360aad926284d544d7049f45189db5664f3c4d07350559e"),
	PowLimit:         newBigIntFromHex("00000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
	CoinbaseMaturity: 240,
	PubKeyHashAddrID: 0x71, // starts with n
	ScriptHashAddrID: 0xc4, // starts with 2
	PrivateKeyID:     0xf1, // starts with 9 or c
	Bech32HRPSegwit:  "tdoge",
	HDCoinType:       1,
}

// DogeRegTestParams defines the network parameters for the regression test Dogecoin network.
var DogeRegTestParams = chaincfg.Params{
	Name:             "regtest",
	Net:              wire.BitcoinNet(0xfabfb5da), // Dogecoin RegTest magic
	DefaultPort:      "18444",
	DNSSeeds:         []chaincfg.DNSSeed{},
	GenesisHash:      newHashFromStr("3d2160a3b5dc4a9d62e7404bb5aa85b0183cd8db1d244508f6003d23713e8819"),
	PowLimit:         newBigIntFromHex("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
	CoinbaseMaturity: 150,
	PubKeyHashAddrID: 0x6f, // starts with m or n
	ScriptHashAddrID: 0xc4, // starts with 2
	PrivateKeyID:     0xef,
	Bech32HRPSegwit:  "rdoge",
	HDCoinType:       1,
}

func newHashFromStr(str string) *chainhash.Hash {
	hash, _ := chainhash.NewHashFromStr(str)
	return hash
}

func newBigIntFromHex(str string) *big.Int {
	i, _ := new(big.Int).SetString(str, 16)
	return i
}
