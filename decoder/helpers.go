package decoder

// NewConfigWithProtocol creates a configuration with the specified protocol ID
func NewConfigWithProtocol(protocolID string) *ParserConfig {
	if protocolID == "" {
		protocolID = "6d6574616964"
	}
	return &ParserConfig{
		ProtocolID:      protocolID,
		CreatorResolver: nil, // Don't resolve creator by default (requires node)
	}
}

// NewConfigWithResolver creates a complete configuration with CreatorResolver
func NewConfigWithResolver(protocolID string, creatorResolver CreatorResolver) *ParserConfig {
	if protocolID == "" {
		protocolID = "6d6574616964"
	}
	return &ParserConfig{
		ProtocolID:      protocolID,
		CreatorResolver: creatorResolver,
	}
}
