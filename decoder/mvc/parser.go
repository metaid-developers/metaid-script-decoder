package mvc

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/bitcoinsv/bsvd/chaincfg"
	"github.com/bitcoinsv/bsvd/txscript"
	"github.com/bitcoinsv/bsvd/wire"

	"github.com/metaid-developers/metaid-script-decoder/decoder"
	"github.com/metaid-developers/metaid-script-decoder/decoder/common"

	chaincfg2 "github.com/btcsuite/btcd/chaincfg"
	txscript2 "github.com/btcsuite/btcd/txscript"
)

// MVCParser is the MVC chain parser
type MVCParser struct {
	config *decoder.ParserConfig
}

// NewMVCParser creates an MVC parser
func NewMVCParser(config *decoder.ParserConfig) *MVCParser {
	if config == nil {
		config = decoder.DefaultConfig()
	}
	return &MVCParser{
		config: config,
	}
}

// GetChainName returns the chain name
func (p *MVCParser) GetChainName() string {
	return "mvc"
}

// ParseTransaction parses an MVC transaction
func (p *MVCParser) ParseTransaction(txBytes []byte, chainParams interface{}) ([]*decoder.Pin, error) {
	// Parse chainParams
	params, ok := chainParams.(*chaincfg.Params)
	if !ok && chainParams != nil {
		return nil, fmt.Errorf("invalid chainParams type for MVC, expected *chaincfg.Params")
	}
	if params == nil {
		params = &chaincfg.MainNetParams
	}

	// Deserialize MVC transaction
	msgTx := wire.NewMsgTx(2)
	if err := msgTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %w", err)
	}

	var pins []*decoder.Pin

	// Calculate MVC transaction hash (may differ from standard)
	txHash, err := p.calculateTxHash(msgTx, txBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate tx hash: %w", err)
	}
	// MVC mainly uses OP_RETURN format
	for i, out := range msgTx.TxOut {
		class, _, _, _ := txscript.ExtractPkScriptAddrs(out.PkScript, params)
		if class.String() == "nonstandard" {
			pin := p.parseOpReturnScript(out.PkScript)
			if pin == nil {
				continue
			}

			// Get PIN owner address
			address, vout, outValue, locationIdx := p.getOwner(msgTx, params)
			if address == "" {
				continue
			}

			pin.Id = fmt.Sprintf("%si%d", txHash, vout)
			pin.TxID = txHash
			pin.Vout = uint32(i)
			pin.OwnerAddress = address
			pin.OwnerMetaId = common.CalculateMetaId(address)
			pin.ChainName = "mvc"
			pin.InscriptionTxIndex = i

			//// PIN location
			pin.Location = fmt.Sprintf("%s:%d:%d", txHash, vout, locationIdx)
			pin.Offset = uint64(vout)
			pin.Output = fmt.Sprintf("%s:%d", txHash, vout)
			pin.OutputValue = outValue
			// pin.Timestamp = msgTx.Timestamp

			pins = append(pins, pin)

			break // Usually only one OP_RETURN
		}
	}

	return pins, nil
}

// parseOpReturnScript parses OP_RETURN scripts
func (p *MVCParser) parseOpReturnScript(pkScript []byte) *decoder.Pin {
	if len(pkScript) < 1 {
		return nil
	}

	// Handle two formats:
	// 1. OP_RETURN ... (direct)
	// 2. OP_0 OP_RETURN ... (with OP_0 prefix)
	offset := 0

	// Check if starts with OP_0
	if pkScript[0] == txscript.OP_0 || pkScript[0] == txscript.OP_FALSE {
		offset = 1
	}

	// Check for OP_RETURN
	if offset >= len(pkScript) || pkScript[offset] != txscript.OP_RETURN {
		return nil
	}

	// Extract data pushes from the script after OP_RETURN
	dataPushes, err := extractDataPushes(pkScript[offset+1:])
	if err != nil || len(dataPushes) == 0 {
		return nil
	}

	// Check protocol ID
	if hex.EncodeToString(dataPushes[0]) != p.config.ProtocolID {
		return nil
	}

	return p.parseOnePin(dataPushes[1:])
}

// parseOnePin parses a single PIN data
func (p *MVCParser) parseOnePin(infoList [][]byte) *decoder.Pin {
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
	pin.OriginalPath = string(infoList[1])

	// Parse host and path from OriginalPath
	// Format can be: "host:path" or just "path"
	// For cases like "example.com:8080:/path", we need to find ":/" pattern
	pathStr := pin.OriginalPath

	// Look for ":/" pattern to identify the separator between host and path
	colonSlashIndex := strings.Index(pathStr, ":/")
	if colonSlashIndex > 0 {
		// Found ":/", split into host and path
		pin.Host = pathStr[:colonSlashIndex]
		pin.Path = common.NormalizePath(pathStr[colonSlashIndex+1:]) // Skip the colon, keep the slash
	} else {
		// No ":/" pattern found, treat entire string as path
		pin.Host = ""
		pin.Path = common.NormalizePath(pathStr)
	}
	pin.ParentPath = pin.Path

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

// getOwner gets the owner of the PIN
func (p *MVCParser) getOwner(tx *wire.MsgTx, params *chaincfg.Params) (address string, vout int, outValue int64, locationIdx int64) {
	for i, out := range tx.TxOut {
		params2 := &chaincfg2.MainNetParams
		if params == &chaincfg.TestNet3Params {
			params2 = &chaincfg2.TestNet3Params
		}
		class, addresses, _, _ := txscript2.ExtractPkScriptAddrs(out.PkScript, params2)
		if class.String() != "nulldata" && class.String() != "nonstandard" && len(addresses) > 0 {
			address = addresses[0].EncodeAddress()
			vout = i
			outValue = out.Value
			locationIdx = 0
			fmt.Println("address", address)
			fmt.Println("vout", vout)
			return
		}
	}
	return "", 0, 0, 0
}

// calculateTxHash calculates the MVC transaction hash
// MVC may use a special transaction hash calculation method
func (p *MVCParser) calculateTxHash(msgTx *wire.MsgTx, txBytes []byte) (string, error) {
	// Serialize transaction
	buffer := new(bytes.Buffer)
	if err := msgTx.Serialize(buffer); err != nil {
		return "", err
	}

	// Parse raw transaction to get version information
	rawTx, err := decodeRawTransaction(buffer.Bytes())
	if err != nil {
		return "", err
	}

	// If version >= 10, use new hash algorithm
	version := binary.LittleEndian.Uint32(rawTx.Version)
	if version < 10 {
		return rawTx.TxID, nil
	}

	// Use new hash algorithm
	newRawTxByte := getTxNewRawByte(rawTx)
	return getTxID(hex.EncodeToString(newRawTxByte)), nil
}

// RawTransaction is the MVC raw transaction structure
type RawTransaction struct {
	TxID     string
	Version  []byte
	Vins     []TxIn
	Vouts    []TxOut
	LockTime []byte
	inSize   uint64
	outSize  uint64
}

// TxIn represents a transaction input
type TxIn struct {
	TxID      []byte
	Vout      []byte
	scriptSig []byte
	sequence  []byte
}

// TxOut represents a transaction output
type TxOut struct {
	amount     []byte
	lockScript []byte
}

// decodeRawTransaction decodes a raw transaction
func decodeRawTransaction(txBytes []byte) (*RawTransaction, error) {
	limit := len(txBytes)
	if limit == 0 {
		return nil, errors.New("invalid transaction data")
	}

	var rawTx RawTransaction
	index := 0

	// Version (4 bytes)
	if index+4 > limit {
		return nil, errors.New("invalid transaction data length")
	}
	rawTx.Version = txBytes[index : index+4]
	index += 4

	// Input count
	icount, length := decodeVarInt(txBytes[index:])
	numOfVins := icount
	rawTx.inSize = uint64(numOfVins)
	index += length

	if numOfVins == 0 {
		return nil, errors.New("invalid transaction data: no inputs")
	}

	// Parse inputs
	for i := 0; i < numOfVins; i++ {
		var tmpTxIn TxIn

		if index+32 > limit {
			return nil, errors.New("invalid transaction data length")
		}
		tmpTxIn.TxID = txBytes[index : index+32]
		index += 32

		if index+4 > limit {
			return nil, errors.New("invalid transaction data length")
		}
		tmpTxIn.Vout = txBytes[index : index+4]
		index += 4

		scriptLen, length := decodeVarInt(txBytes[index:])
		index += length

		tmpTxIn.scriptSig = txBytes[index : index+scriptLen]
		index += scriptLen

		tmpTxIn.sequence = txBytes[index : index+4]
		index += 4
		rawTx.Vins = append(rawTx.Vins, tmpTxIn)
	}

	// Output count
	icount, length = decodeVarInt(txBytes[index:])
	numOfVouts := icount
	rawTx.outSize = uint64(numOfVouts)
	index += length

	if numOfVouts == 0 {
		return nil, errors.New("invalid transaction data: no outputs")
	}

	// Parse outputs
	for i := 0; i < numOfVouts; i++ {
		var tmpTxOut TxOut

		if index+8 > limit {
			return nil, errors.New("invalid transaction data length")
		}
		tmpTxOut.amount = txBytes[index : index+8]
		index += 8

		lockScriptLen, length := decodeVarInt(txBytes[index:])
		index += length

		if lockScriptLen == 0 {
			return nil, errors.New("invalid transaction data: empty lockScript")
		}
		if index+lockScriptLen > limit {
			return nil, errors.New("invalid transaction data length")
		}
		tmpTxOut.lockScript = txBytes[index : index+lockScriptLen]
		index += lockScriptLen
		rawTx.Vouts = append(rawTx.Vouts, tmpTxOut)
	}

	// LockTime (4 bytes)
	if index+4 > limit {
		return nil, errors.New("invalid transaction data length")
	}
	rawTx.LockTime = txBytes[index : index+4]
	index += 4

	if index != limit {
		return nil, errors.New("too much transaction data")
	}

	// Calculate TxID
	if uint64(binary.LittleEndian.Uint32(rawTx.Version)) < 10 {
		rawTx.TxID = getTxID(hex.EncodeToString(txBytes))
	} else {
		newRawTxByte := getTxNewRawByte(&rawTx)
		rawTx.TxID = getTxID(hex.EncodeToString(newRawTxByte))
	}

	return &rawTx, nil
}

// decodeVarInt decodes a variable-length integer
func decodeVarInt(buf []byte) (int, int) {
	if len(buf) == 0 {
		return 0, 0
	}

	if buf[0] <= 0xfc {
		return int(buf[0]), 1
	} else if buf[0] == 0xfd {
		if len(buf) < 3 {
			return 0, 0
		}
		return (int(buf[2]) * int(math.Pow(256, 1))) + int(buf[1]), 3
	} else if buf[0] == 0xfe {
		if len(buf) < 5 {
			return 0, 0
		}
		count := (int(buf[4]) * int(math.Pow(256, 3))) +
			(int(buf[3]) * int(math.Pow(256, 2))) +
			(int(buf[2]) * int(math.Pow(256, 1))) +
			int(buf[1])
		return count, 5
	} else if buf[0] == 0xff {
		if len(buf) < 9 {
			return 0, 0
		}
		count := (int(buf[8]) * int(math.Pow(256, 7))) +
			int(buf[7])*int(math.Pow(256, 6)) +
			int(buf[6])*int(math.Pow(256, 5)) +
			int(buf[5])*int(math.Pow(256, 4)) +
			int(buf[4])*int(math.Pow(256, 3)) +
			int(buf[3])*int(math.Pow(256, 2)) +
			int(buf[2])*int(math.Pow(256, 1)) +
			int(buf[1])
		return count, 9
	}
	return 0, 0
}

// getTxID calculates the transaction ID
func getTxID(hexString string) string {
	code, _ := hex.DecodeString(hexString)
	dHash := doubleHashB(code)
	return hex.EncodeToString(reverseBytes(dHash))
}

// doubleHashB calculates double SHA256
func doubleHashB(b []byte) []byte {
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	return second[:]
}

// reverseBytes reverses a byte array
func reverseBytes(s []byte) []byte {
	result := make([]byte, len(s))
	copy(result, s)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// uint32ToLittleEndianBytes converts uint32 to little-endian bytes
func uint32ToLittleEndianBytes(data uint32) []byte {
	tmp := [4]byte{}
	binary.LittleEndian.PutUint32(tmp[:], data)
	return tmp[:]
}

// sha256Hash calculates SHA256 hash
func sha256Hash(message []byte) []byte {
	hash := sha256.New()
	hash.Write(message)
	return hash.Sum(nil)
}

// extractDataPushes extracts data pushes from a script
func extractDataPushes(script []byte) ([][]byte, error) {
	var dataPushes [][]byte
	offset := 0

	for offset < len(script) {
		if offset >= len(script) {
			break
		}

		opcode := script[offset]
		offset++

		var data []byte
		var dataLen int

		// Handle different opcodes
		if opcode <= txscript.OP_PUSHDATA4 {
			if opcode < txscript.OP_PUSHDATA1 {
				// Direct push (0-75 bytes)
				dataLen = int(opcode)
			} else if opcode == txscript.OP_PUSHDATA1 {
				// Next byte is the length
				if offset >= len(script) {
					return nil, errors.New("script truncated")
				}
				dataLen = int(script[offset])
				offset++
			} else if opcode == txscript.OP_PUSHDATA2 {
				// Next 2 bytes are the length (little-endian)
				if offset+1 >= len(script) {
					return nil, errors.New("script truncated")
				}
				dataLen = int(binary.LittleEndian.Uint16(script[offset : offset+2]))
				offset += 2
			} else if opcode == txscript.OP_PUSHDATA4 {
				// Next 4 bytes are the length (little-endian)
				if offset+3 >= len(script) {
					return nil, errors.New("script truncated")
				}
				dataLen = int(binary.LittleEndian.Uint32(script[offset : offset+4]))
				offset += 4
			}

			// Extract the data
			if offset+dataLen > len(script) {
				return nil, errors.New("script truncated")
			}
			data = make([]byte, dataLen)
			copy(data, script[offset:offset+dataLen])
			offset += dataLen

			dataPushes = append(dataPushes, data)
		} else {
			// For other opcodes, we skip them or could handle specially
			// For OP_RETURN parsing, we typically only care about data pushes
			continue
		}
	}

	return dataPushes, nil
}

// getTxNewRawByte gets new transaction bytes (for transactions with version >= 10)
func getTxNewRawByte(transaction *RawTransaction) []byte {
	var (
		newRawTxByte   []byte
		newInputsByte  []byte
		newInputs2Byte []byte
		newOutputsByte []byte
	)

	newRawTxByte = append(newRawTxByte, transaction.Version...)
	newRawTxByte = append(newRawTxByte, transaction.LockTime...)
	newRawTxByte = append(newRawTxByte, uint32ToLittleEndianBytes(uint32(transaction.inSize))...)
	newRawTxByte = append(newRawTxByte, uint32ToLittleEndianBytes(uint32(transaction.outSize))...)

	for _, in := range transaction.Vins {
		newInputsByte = append(newInputsByte, in.TxID...)
		newInputsByte = append(newInputsByte, in.Vout...)
		newInputsByte = append(newInputsByte, in.sequence...)

		newInputs2Byte = append(newInputs2Byte, sha256Hash(in.scriptSig)...)
	}
	newRawTxByte = append(newRawTxByte, sha256Hash(newInputsByte)...)
	newRawTxByte = append(newRawTxByte, sha256Hash(newInputs2Byte)...)

	for _, out := range transaction.Vouts {
		newOutputsByte = append(newOutputsByte, out.amount...)
		newOutputsByte = append(newOutputsByte, sha256Hash(out.lockScript)...)
	}
	newRawTxByte = append(newRawTxByte, sha256Hash(newOutputsByte)...)

	return newRawTxByte
}

func PkScriptToAddress(net *chaincfg.Params, pkScript string) (string, error) {
	pkScriptByte, err := hex.DecodeString(pkScript)
	if err != nil {
		return "", err
	}
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScriptByte, net)
	if err != nil {
		return "", errors.New("Extract address from pkScript. ")
	}
	if len(addrs) == 0 {
		return "", errors.New("Extract address from pkScript. ")
	}
	address := addrs[0].EncodeAddress()
	return address, nil
}

func PkScriptToAddres2(net *chaincfg2.Params, pkScript string) (string, error) {
	pkScriptByte, err := hex.DecodeString(pkScript)
	if err != nil {
		return "", err
	}
	_, addrs, _, err := txscript2.ExtractPkScriptAddrs(pkScriptByte, net)
	if err != nil {
		return "", errors.New("Extract address from pkScript. ")
	}
	if len(addrs) == 0 {
		return "", errors.New("Extract address from pkScript. ")
	}
	address := addrs[0].EncodeAddress()
	return address, nil
}
