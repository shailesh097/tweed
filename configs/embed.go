// Package configs embeds the default tweed.yaml into the binary, so a freshly
// downloaded tweed works without cloning the repository.
package configs

import _ "embed"

// Default is the contents of configs/tweed.yaml at build time.
//
//go:embed tweed.yaml
var Default []byte
