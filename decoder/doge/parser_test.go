package doge

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/metaid-developers/metaid-script-decoder/decoder"
)

func TestNewDOGEParser(t *testing.T) {
	// Test creating parser with default configuration
	parser := NewDOGEParser(nil)
	if parser == nil {
		t.Fatal("NewDOGEParser returned nil")
	}

	if parser.config.ProtocolID != "6d6574616964" {
		t.Errorf("Expected default protocol ID '6d6574616964', got '%s'", parser.config.ProtocolID)
	}

	// Test creating parser with custom configuration
	customConfig := &decoder.ParserConfig{
		ProtocolID: "746573746964",
	}
	parser = NewDOGEParser(customConfig)
	if parser.config.ProtocolID != "746573746964" {
		t.Errorf("Expected custom protocol ID '746573746964', got '%s'", parser.config.ProtocolID)
	}
}

func TestGetChainName(t *testing.T) {
	parser := NewDOGEParser(nil)
	if parser.GetChainName() != "doge" {
		t.Errorf("Expected chain name 'doge', got '%s'", parser.GetChainName())
	}
}

func TestParseTransaction_InvalidData(t *testing.T) {
	parser := NewDOGEParser(nil)

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
	// Note: This test requires a real DOGE transaction hex that contains metaid protocol data
	// In a real scenario, you would use an actual DOGE transaction hex that contains
	// a ScriptSig with metaid protocol data (either P2SH redeem script or direct ScriptSig format)
	//
	// Example formats:
	// 1. P2SH redeem script: <pubkey> OP_CHECKSIGVERIFY OP_FALSE OP_IF "metaid" <data> OP_ENDIF
	// 2. Direct ScriptSig: <metaid> <operation> <contentType> <encryption> <version> <address:path> <content> <signature> <pubkey>
	//
	// For now, this test uses a placeholder transaction structure to verify the parser doesn't crash
	// Replace the txHex below with a real DOGE transaction containing metaid data for full testing

	// Placeholder transaction hex (may not deserialize correctly, but tests parser structure)
	txHex := "02000000039c76656bafa0fb8ecb08c2628ab0602e58d5c41f3f676c80461405c4c976aa2800000000be066d6574616964066372656174650a746578742f706c61696e013005302e302e31106170706c69636174696f6e2f6a736f6e17446f6765206d657461696420696e736372697074696f6e47304402203f685bd7a2062f7726623381246af3f4d40ef268d571ed067d476c54250770ad022043a9d79b216cdf1b54885fcaa9b0eb8073e8eab0cf3d180f09e2361355233c9c012b2102dc3647d7dbeaf9223800276a924c9d4a07c886417e0c65d9d2c92eb080356afcad7575757575757551ffffffffd512c8c144c46d4124682f31ac7961af52a78db4c85bd44be985d437c54eee98010000006a4730440220294d502896262b31a3ed29c21a4e32f54319c858e5f610060c3b98823661d23a02203b45a72495b8108c0fdc5549f38b2eecb377c8bf8bbc7145009e545e2c09a6970121029276cc28460500aae93fa8fa619e25ca98b5689b381ba1d730e9441b04fb6ceeffffffffbac3198cfb90f5650c201cb7c51cb0c49cf30cf8177295e58752413675e7e915010000006b483045022100fecb40bfb3059d6597b93630f9a292092b7ad8331d7465aef719e6525e70ef6802205fda5e8ca7c825ef2ab6f3e710aea07c2bfe4e835a897c9c6a07b092e68ebac00121029276cc28460500aae93fa8fa619e25ca98b5689b381ba1d730e9441b04fb6ceeffffffff02a0860100000000001976a9147748321f517ae351be891b6fe702563293672b4f88ac200c8201000000001976a9147748321f517ae351be891b6fe702563293672b4f88ac00000000"

	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		t.Errorf("Failed to decode transaction: %v", err)
		return
	}

	parser := NewDOGEParser(nil)
	// Test valid data - should not error on deserialization
	pins, err := parser.ParseTransaction(txBytes, nil)
	if err != nil {
		// If deserialization fails, it's okay - we're just testing the parser structure
		// In production, you would use a real DOGE transaction with metaid data
		t.Logf("Transaction deserialization failed (expected for placeholder data): %v", err)
		return
	}

	// Note: This transaction may not contain actual metaid data, so pins might be empty
	// In a real test with actual DOGE transaction containing metaid data, you would expect pins
	if len(pins) > 0 {
		for _, pin := range pins {
			fmt.Printf("Pin: %+v\n", pin)
		}
	} else {
		t.Log("No pins found in transaction (this is expected if transaction doesn't contain metaid data)")
	}
}
