package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/metaid-developers/metaid-script-decoder/decoder"
	"github.com/metaid-developers/metaid-script-decoder/decoder/btc"
	"github.com/metaid-developers/metaid-script-decoder/decoder/common"

	"github.com/btcsuite/btcd/chaincfg"
)

// Example: How to use CreatorResolver and MetaIdCalculator

// SimpleCreatorResolver is a simple example implementation of creator resolver
// In actual use, you need to connect to a real blockchain node
type SimpleCreatorResolver struct {
	// You can add node connection configuration here
	// nodeClient *YourNodeClient
}

// ResolveCreator implements the CreatorResolver interface
// Query creator address based on CreatorInputLocation (txId:vout format)
func (r *SimpleCreatorResolver) ResolveCreator(chainName, txId string, vout uint32) (string, string, error) {
	// Actual implementation needs to:
	// 1. Use chainName to select the correct node
	// 2. Query the blockchain node to get the transaction based on txId and vout
	// 3. Get the address of the specified vout from the transaction output
	// 4. Calculate MetaID

	// Example code:
	// address := r.queryAddressFromNode(chainName, txId, vout)
	// metaId := common.CalculateMetaId(address)

	// Return example values temporarily
	exampleAddress := "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"
	exampleMetaId := common.CalculateMetaId(exampleAddress)

	return exampleAddress, exampleMetaId, nil
}

func ExampleWithResolver() {
	fmt.Println("=== Example: Using CreatorResolver and MetaIdCalculator ===")

	// 1. Create custom CreatorResolver
	resolver := &SimpleCreatorResolver{}

	// 2. Create configuration with resolver
	config := decoder.NewConfigWithResolver(
		"6d6574616964", // metaid protocol
		resolver,       // creator resolver
	)

	// 3. Create parser
	parser := btc.NewBTCParser(config)

	// 4. Parse transaction
	txHex := "your_transaction_hex_here"
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		log.Printf("Failed to decode transaction: %v", err)
		return
	}

	pins, err := parser.ParseTransaction(txBytes, &chaincfg.MainNetParams)
	if err != nil {
		log.Printf("Failed to parse transaction: %v", err)
		return
	}

	// 5. Output results
	fmt.Printf("Found %d PIN(s):\n", len(pins))
	for i, pin := range pins {
		fmt.Printf("\nPIN #%d:\n", i+1)
		fmt.Printf("  ID: %s\n", pin.Id)
		fmt.Printf("  Owner: %s\n", pin.OwnerAddress)
		fmt.Printf("  Owner MetaID: %s\n", pin.OwnerMetaId)
		fmt.Printf("  Creator: %s\n", pin.CreatorAddress)
		fmt.Printf("  Creator MetaID: %s\n", pin.CreatorMetaId)
		fmt.Printf("  Creator Input: %s\n", pin.CreatorInputLocation)
		fmt.Printf("  Operation: %s\n", pin.Operation)
		fmt.Printf("  Path: %s\n", pin.Path)
	}
}

func ExampleWithoutResolver() {
	fmt.Println("=== Example: Without CreatorResolver ===")

	// Use default configuration: don't query creator
	// Note: MetaID will be calculated automatically (built-in feature)
	config := decoder.DefaultConfig()

	parser := btc.NewBTCParser(config)

	txHex := "your_transaction_hex_here"
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		log.Printf("Failed to decode transaction: %v", err)
		return
	}

	pins, err := parser.ParseTransaction(txBytes, &chaincfg.MainNetParams)
	if err != nil {
		log.Printf("Failed to parse transaction: %v", err)
		return
	}

	// Output results
	fmt.Printf("Found %d PIN(s):\n", len(pins))
	for i, pin := range pins {
		fmt.Printf("\nPIN #%d:\n", i+1)
		fmt.Printf("  Owner: %s\n", pin.OwnerAddress)
		fmt.Printf("  Owner MetaID: %s (automatically calculated)\n", pin.OwnerMetaId)
		fmt.Printf("  Creator Input Location: %s\n", pin.CreatorInputLocation)
		fmt.Printf("  Note: Creator address is empty (requires CreatorResolver configuration)\n")
	}
}

func ExampleMinimal() {
	fmt.Println("=== Example: Minimal Configuration ===")

	// Use default configuration: don't query creator
	// MetaID will be calculated automatically (sha256(address) is a built-in feature)
	config := decoder.DefaultConfig()

	parser := btc.NewBTCParser(config)

	txHex := "your_transaction_hex_here"
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		log.Printf("Failed to decode transaction: %v", err)
		return
	}

	pins, err := parser.ParseTransaction(txBytes, &chaincfg.MainNetParams)
	if err != nil {
		log.Printf("Failed to parse transaction: %v", err)
		return
	}

	// Output results
	fmt.Printf("Found %d PIN(s):\n", len(pins))
	for i, pin := range pins {
		fmt.Printf("\nPIN #%d:\n", i+1)
		fmt.Printf("  Owner: %s\n", pin.OwnerAddress)
		fmt.Printf("  Owner MetaID: %s (automatically calculated)\n", pin.OwnerMetaId)
		fmt.Printf("  Creator Input Location: %s\n", pin.CreatorInputLocation)
		fmt.Printf("  Operation: %s\n", pin.Operation)
		fmt.Printf("  Path: %s\n", pin.Path)
		fmt.Printf("  Note: Creator address is empty (CreatorResolver not configured)\n")
	}
}

// Main function demonstrates different usage methods
func mainWithResolverExample() {
	// Method 1: Full functionality (with CreatorResolver)
	ExampleWithResolver()

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Method 2: Only calculate MetaID
	ExampleWithoutResolver()

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Method 3: Minimal configuration
	ExampleMinimal()
}
