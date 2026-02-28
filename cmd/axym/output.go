package main

import (
	"encoding/json"
	"fmt"
	"io"
)

type globalFlags struct {
	JSON    bool
	Quiet   bool
	Explain bool
}

type envelope struct {
	OK      bool           `json:"ok"`
	Command string         `json:"command"`
	Data    any            `json:"data,omitempty"`
	Error   *errorEnvelope `json:"error,omitempty"`
}

type errorEnvelope struct {
	Reason     string `json:"reason"`
	Message    string `json:"message"`
	BreakIndex int    `json:"break_index"`
	BreakPoint string `json:"break_point,omitempty"`
}

func printJSON(w io.Writer, payload any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func printText(w io.Writer, text string, quiet bool) {
	if quiet {
		return
	}
	_, _ = fmt.Fprintln(w, text)
}
