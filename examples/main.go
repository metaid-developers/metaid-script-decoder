package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"metaid-script-decoder/decoder"
	"metaid-script-decoder/decoder/btc"
	"metaid-script-decoder/decoder/mvc"

	"github.com/btcsuite/btcd/chaincfg"
)

func main() {
	fmt.Println("=== MetaID Script Decoder Examples ===")

	// Example 1: Parse BTC transaction
	fmt.Println("Example 1: Parse BTC Transaction")
	parseBTCTransaction()

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 2: Parse MVC transaction
	fmt.Println("Example 2: Parse MVC Transaction")
	parseMVCTransaction()

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 3: Use custom protocol ID
	fmt.Println("Example 3: Use Custom Protocol ID")
	parseWithCustomProtocolID()
}

// parseBTCTransaction example of parsing BTC transactions
func parseBTCTransaction() {
	// This is a sample transaction hex string
	// In actual use, you need to get real transaction data from the blockchain node
	txHex := "your_btc_transaction_hex_here"

	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		log.Printf("Failed to decode transaction: %v", err)
		return
	}

	// Create BTC parser
	parser := btc.NewBTCParser(nil)

	// Parse transaction
	pins, err := parser.ParseTransaction(txBytes, &chaincfg.MainNetParams)
	if err != nil {
		log.Printf("Failed to parse transaction: %v", err)
		return
	}

	// Output results
	fmt.Printf("Found %d PIN(s):\n", len(pins))
	for i, pin := range pins {
		printPin(i+1, pin)
	}
}

// parseMVCTransaction example of parsing MVC transactions
func parseMVCTransaction() {
	// This is a sample transaction hex string
	txHex := "your_mvc_transaction_hex_here"

	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		log.Printf("Failed to decode transaction: %v", err)
		return
	}

	// Create MVC parser
	parser := mvc.NewMVCParser(nil)

	// Parse transaction
	pins, err := parser.ParseTransaction(txBytes, &chaincfg.MainNetParams)
	if err != nil {
		log.Printf("Failed to parse transaction: %v", err)
		return
	}

	// Output results
	fmt.Printf("Found %d PIN(s):\n", len(pins))
	for i, pin := range pins {
		printPin(i+1, pin)
	}
}

// parseWithCustomProtocolID parsing with custom protocol ID
func parseWithCustomProtocolID() {
	txHex := "your_transaction_hex_here"

	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		log.Printf("Failed to decode transaction: %v", err)
		return
	}

	// Create custom configuration
	config := &decoder.ParserConfig{
		ProtocolID: "6d6574616964", // hex of metaid
	}

	// Create parser with custom configuration
	parser := btc.NewBTCParser(config)

	// Parse transaction
	pins, err := parser.ParseTransaction(txBytes, &chaincfg.TestNet3Params)
	if err != nil {
		log.Printf("Failed to parse transaction: %v", err)
		return
	}

	// Output results
	fmt.Printf("Found %d PIN(s):\n", len(pins))
	for i, pin := range pins {
		printPin(i+1, pin)
	}
}

// printPin prints PIN information
func printPin(index int, pin *decoder.Pin) {
	fmt.Printf("\nPIN #%d:\n", index)
	fmt.Printf("  Chain: %s\n", pin.ChainName)
	fmt.Printf("  TxID: %s\n", pin.TxID)
	fmt.Printf("  Output Index: %d\n", pin.Vout)
	fmt.Printf("  Operation: %s\n", pin.Operation)
	fmt.Printf("  Path: %s\n", pin.Path)
	fmt.Printf("  Parent Path: %s\n", pin.ParentPath)
	fmt.Printf("  Content Type: %s\n", pin.ContentType)
	fmt.Printf("  Content Length: %d bytes\n", pin.ContentLength)
	fmt.Printf("  Version: %s\n", pin.Version)
	fmt.Printf("  Encryption: %s\n", pin.Encryption)
	fmt.Printf("  Owner Address: %s\n", pin.OwnerAddress)
	fmt.Printf("  Owner MetaID: %s\n", pin.OwnerMetaId)

	// If it's JSON content, try to format the output
	if pin.ContentType == "application/json" && len(pin.ContentBody) > 0 {
		var jsonData interface{}
		if err := json.Unmarshal(pin.ContentBody, &jsonData); err == nil {
			prettyJSON, _ := json.MarshalIndent(jsonData, "  ", "  ")
			fmt.Printf("  Content:\n  %s\n", string(prettyJSON))
		} else {
			fmt.Printf("  Content: %s\n", string(pin.ContentBody))
		}
	} else if len(pin.ContentBody) > 0 && len(pin.ContentBody) < 200 {
		fmt.Printf("  Content: %s\n", string(pin.ContentBody))
	}
}
