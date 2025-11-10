package decoder

// Pin represents the PIN data structure in the MetaID protocol
type Pin struct {
	Id string `json:"id"` // PIN ID

	// PIN owner
	OwnerAddress string `json:"ownerAddress"` // Owner address
	OwnerMetaId  string `json:"ownerMetaId"`  // Owner MetaID
	// PIN creator
	CreatorAddress       string `json:"creatorAddress"`       // Creator address
	CreatorMetaId        string `json:"creatorMetaId"`        // Creator MetaID
	CreatorInputLocation string `json:"creatorInputLocation"` // Creator input location txId:vin

	// PIN location
	Offset      uint64 `json:"offset"`
	Location    string `json:"location"`
	Output      string `json:"output"`
	OutputValue int64  `json:"outputValue"`
	Timestamp   int64  `json:"timestamp"`

	// Basic fields
	Operation  string `json:"operation"`  // Operation type: create, modify, revoke, etc.
	Path       string `json:"path"`       // PIN path
	ParentPath string `json:"parentPath"` // Parent path
	Encryption string `json:"encryption"` // Encryption method
	Version    string `json:"version"`    // Version

	// Content fields
	ContentType   string `json:"contentType"`   // Content type
	ContentBody   []byte `json:"contentBody"`   // Content body
	ContentLength uint64 `json:"contentLength"` // Content length

	// Blockchain-related fields
	TxID string `json:"txId"` // Transaction ID
	Vout uint32 `json:"vout"` // Output index

	// Parsing metadata
	ChainName          string `json:"chainName"`          // Chain name: btc, mvc, etc.
	InscriptionTxIndex int    `json:"inscriptionTxIndex"` // Index position in transaction
}

// ChainParser is the interface for chain parsers
type ChainParser interface {
	// ParseTransaction parses PIN data from transaction bytes
	ParseTransaction(txBytes []byte, chainParams interface{}) ([]*Pin, error)

	// GetChainName returns the chain name
	GetChainName() string
}

// CreatorResolver is the interface for creator address resolver
// External implementations can provide node query functionality
type CreatorResolver interface {
	// ResolveCreator resolves the creator address based on txId and vout
	// Returns: creatorAddress, creatorMetaId, error
	ResolveCreator(chainName, txId string, vout uint32) (string, string, error)
}

// ParserConfig represents the parser configuration
type ParserConfig struct {
	ProtocolID string // Protocol ID as hex string, default is "6d6574616964" (metaid)

	// CreatorResolver is an optional creator address resolver
	// If not provided, CreatorAddress and CreatorMetaId will be empty
	CreatorResolver CreatorResolver
}

// DefaultConfig returns the default configuration
func DefaultConfig() *ParserConfig {
	return &ParserConfig{
		ProtocolID:      "6d6574616964", // metaid
		CreatorResolver: nil,             // Don't resolve creator by default
	}
}
