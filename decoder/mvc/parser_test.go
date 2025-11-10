package mvc

import (
	"testing"

	"metaid-script-decoder/decoder"
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
