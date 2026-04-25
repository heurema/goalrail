package term

import (
	"encoding/json"
	"fmt"
	"io"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Output contains terminal writers for command packages.
type Output struct {
	Stdout io.Writer
	Stderr io.Writer
}

func New(stdout, stderr io.Writer) *Output {
	return &Output{Stdout: stdout, Stderr: stderr}
}

func ParseFormat(value string) (Format, error) {
	switch Format(value) {
	case "", FormatText:
		return FormatText, nil
	case FormatJSON:
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("unsupported format %q; expected text or json", value)
	}
}

func WriteJSON(w io.Writer, v any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
