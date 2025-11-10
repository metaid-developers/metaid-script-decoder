package decoder

// Note: Factory methods have been removed to avoid circular imports
// Please use each chain's parser directly:
//
// For BTC:
//   import "github.com/metaid-developers/metaid-script-decoder/decoder/btc"
//   parser := btc.NewBTCParser(config)
//
// For MVC:
//   import "github.com/metaid-developers/metaid-script-decoder/decoder/mvc"
//   parser := mvc.NewMVCParser(config)
//
// For example usage, see examples/main.go
