//go:build tools
// +build tools

// Package tools tracks development-only dependencies (those that the
// production binary doesn't import) so `go mod tidy` doesn't drop them.
// See https://github.com/go-modules/by-example/blob/master/tools/README.md.
package tools

import (
	_ "ariga.io/atlas-provider-gorm/gormschema"
)