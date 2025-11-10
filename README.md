# MetaID Script Decoder

Chinese README: [README-ZH.md](README-ZH.md)

A pure Go parsing tool library for the MetaID protocol, used to extract and parse PIN (Personal Information Node) data from blockchain transactions.

## Supported Chains

| Chain Name | Identifier | Supported Formats |
|--------|--------|------------|
| Bitcoin | `btc`, `bitcoin` | Witness (OP_FALSE + OP_IF), OP_RETURN |
| MicroVisionChain | `mvc`, `microvisionchain` | OP_RETURN |


## Quick Start

### Basic Usage - BTC Chain

```go
package main

import (
    "encoding/hex"
    "fmt"
    "log"

    "github.com/btcsuite/btcd/chaincfg"
    "metaid-script-decoder/decoder/btc"
)

func main() {
    // Transaction hex string
    txHex := "your_transaction_hex_here"
    txBytes, _ := hex.DecodeString(txHex)

    // Create BTC parser
    parser := btc.NewBTCParser(nil)

    // Parse transaction
    pins, err := parser.ParseTransaction(txBytes, &chaincfg.MainNetParams)
    if err != nil {
        log.Fatal(err)
    }

    // Output results
    for _, pin := range pins {
        fmt.Printf("Operation: %s, Path: %s\n", pin.Operation, pin.Path)
        fmt.Printf("Content: %s\n", string(pin.ContentBody))
    }
}
```

### Basic Usage - MVC Chain

```go
import (
    "metaid-script-decoder/decoder/mvc"
)

// Create MVC parser
parser := mvc.NewMVCParser(nil)

// Parse transaction
pins, err := parser.ParseTransaction(txBytes, &chaincfg.MainNetParams)
```

### Using Custom Protocol ID

```go
import (
    "metaid-script-decoder/decoder"
    "metaid-script-decoder/decoder/btc"
)

// Create custom configuration
config := &decoder.ParserConfig{
    ProtocolID: "6d6574616964", // hex of metaid
}

// Create parser with custom configuration
parser := btc.NewBTCParser(config)

// Parse transaction
pins, err := parser.ParseTransaction(txBytes, &chaincfg.TestNet3Params)
```

## PIN Data Structure

```go
type Pin struct {
    // Basic fields
    Operation  string // Operation type: create, modify, revoke
    Path       string // PIN path
    ParentPath string // Parent path

    // Content fields
    ContentType   string // Content type
    ContentBody   []byte // Content body
    ContentLength uint64 // Content length

    // Metadata
    Encryption string // Encryption method
    Version    string // Version

    // Blockchain-related fields
    TxID    string // Transaction ID
    Vout    uint32 // Output index
    Address string // Address (PIN owner)

    // Parsing metadata
    ChainName string // Chain name: btc, mvc
    TxIndex   int    // Index position in transaction
}
```

## MetaID Protocol Description

The MetaID protocol defines a standard format for storing personal information on the blockchain. PIN (Personal Information Node) is the core data structure of the protocol.

### PIN Format

#### Witness Format (BTC)
```
OP_FALSE OP_IF <protocol_id> <operation> <path> <encryption> <version> <content_type> <payload> OP_ENDIF
```

#### OP_RETURN Format (BTC/MVC)
```
OP_RETURN <protocol_id> <operation> <path> <encryption> <version> <content_type> <payload>
```

### Field Description

- **protocol_id**: Protocol identifier (default: `6d6574616964` = "metaid")
- **operation**: Operation type
  - `create`: Create new PIN
  - `modify`: Modify PIN
  - `revoke`: Revoke PIN
- **path**: PIN path, e.g., `/protocols/simplebuzz`
- **encryption**: Encryption method (default: `0` = unencrypted)
- **version**: Version number (default: `0`)
- **content_type**: Content type (default: `application/json`)
- **payload**: Content body (may span multiple fields)

## Extending Support for New Chains

To add support for a new blockchain, simply implement the `ChainParser` interface:

```go
type ChainParser interface {
    ParseTransaction(txBytes []byte, chainParams interface{}) ([]*Pin, error)
    GetChainName() string
}
```

Example:

```go
package mychain

import "metaid-script-decoder/decoder"

type MyChainParser struct {
    config *decoder.ParserConfig
}

func NewMyChainParser(config *decoder.ParserConfig) *MyChainParser {
    return &MyChainParser{config: config}
}

func (p *MyChainParser) GetChainName() string {
    return "mychain"
}

func (p *MyChainParser) ParseTransaction(txBytes []byte, chainParams interface{}) ([]*decoder.Pin, error) {
    // Implement parsing logic
    return pins, nil
}
```

## License

This project uses the same license as the original project. See the [LICENSE](LICENSE) file for details.

## Contributing

Issues and Pull Requests are welcome!

## Related Links

- [MetaID Protocol Documentation](https://metaid.io)

