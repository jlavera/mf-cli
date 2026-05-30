// Package rule provides the embedded LLM/agent usage guide for mf.
package rule

import _ "embed"

// Content is the ready-to-paste rule that teaches an AI agent how to use mf
// in a project. It is surfaced to users via the `mf rule` command.
//
//go:embed content.md
var Content string
