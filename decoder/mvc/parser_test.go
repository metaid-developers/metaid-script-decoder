package mvc

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/metaid-developers/metaid-script-decoder/decoder"
)

func TestNewMVCParser(t *testing.T) {
	// Test creating parser with default configuration
	parser := NewMVCParser(nil)
	if parser == nil {
		t.Fatal("NewMVCParser returned nil")
	}

	if parser.config.ProtocolID != "6d6574616964" {
		t.Errorf("Expected default protocol ID '6d6574616964', got '%s'", parser.config.ProtocolID)
	}

	// Test creating parser with custom configuration
	customConfig := &decoder.ParserConfig{
		ProtocolID: "746573746964",
	}
	parser = NewMVCParser(customConfig)
	if parser.config.ProtocolID != "746573746964" {
		t.Errorf("Expected custom protocol ID '746573746964', got '%s'", parser.config.ProtocolID)
	}
}

func TestGetChainName(t *testing.T) {
	parser := NewMVCParser(nil)
	if parser.GetChainName() != "mvc" {
		t.Errorf("Expected chain name 'mvc', got '%s'", parser.GetChainName())
	}
}

func TestParseTransaction_InvalidData(t *testing.T) {
	parser := NewMVCParser(nil)

	// Test empty data
	_, err := parser.ParseTransaction([]byte{}, nil)
	if err == nil {
		t.Error("Expected error for empty transaction data, got nil")
	}

	// Test invalid data
	_, err = parser.ParseTransaction([]byte{0x01, 0x02, 0x03}, nil)
	if err == nil {
		t.Error("Expected error for invalid transaction data, got nil")
	}
}

func TestParseTransaction_ValidData(t *testing.T) {
	txHex := "0a000000014e581adb0f1856ab2ea847524d621d49ccfe38235ca205c6549caf2370ce5c55020000006a47304402207adb51a78a4f94ab20d001abb44d09272109f465c67443b7b428703b950c6e0502204f952e30d09f64a998237efc79cb44b5da7ea160c56c3c776a07bfdb629bf4f94121039722240e7b2cf378bdc4dc4a0bfd03d2e97e53a674a46229c82b2d9fea2702b9ffffffff0301000000000000001976a914fb6fcbce3e44c49f4037d83a2d7b9a40bdcfdab588ac0000000000000000fd7701006a066d6574616964066372656174654c546263317032306b33783263346d676c6678723577613573677467656368777374706c6438306b727532636734676d6d3475727675617171737661707875303a2f70726f746f636f6c732f73696d706c6562757a7a013005312e302e3010746578742f706c61696e3b7574662d384cf67b22636f6e74656e74223a224d79206e657720706c616e742069732063616c6c6564206120275a5a20506c616e74272062656361757365206974277320737570706f73656420746f20626520696d706f737369626c6520746f206b696c6c2e204368616c6c656e67652061636365707465642e20492063616e206665656c206974206a756467696e67206d6520776974682069747320776178792c20696e646573747275637469626c65206c65617665732e20f09f8cbf2023506c616e744d6f6d2023426c61636b5468756d62222c22636f6e74656e7454797065223a226170706c69636174696f6e2f6a736f6e3b7574662d38227da1a87d06000000001976a914fb6fcbce3e44c49f4037d83a2d7b9a40bdcfdab588ac00000000"
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		t.Errorf("Failed to decode transaction: %v", err)
		return
	}
	parser := NewMVCParser(nil)
	// Test valid data
	pins, err := parser.ParseTransaction(txBytes, nil)
	if err != nil {
		t.Errorf("Expected no error for valid transaction data, got '%s'", err)
		return
	}
	if len(pins) == 0 {
		t.Error("Expected at least one pin, got none")
		return
	}
	for _, pin := range pins {
		fmt.Printf("Pin: %+v\n", pin)
	}
}
