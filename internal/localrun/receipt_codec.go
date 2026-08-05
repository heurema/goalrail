package localrun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const MaxTerminalReceiptBytes = 1 << 20

func DecodeTerminalReceipt(reader io.Reader) (TerminalReceipt, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, MaxTerminalReceiptBytes+1))
	if err != nil {
		return TerminalReceipt{}, fmt.Errorf("read terminal receipt: %w", err)
	}
	if len(raw) > MaxTerminalReceiptBytes {
		return TerminalReceipt{}, fmt.Errorf("terminal receipt exceeds %d bytes", MaxTerminalReceiptBytes)
	}
	var receipt TerminalReceipt
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return TerminalReceipt{}, fmt.Errorf("decode terminal receipt: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return TerminalReceipt{}, fmt.Errorf("decode terminal receipt: trailing JSON")
	}
	if err := validateTerminalReceipt(receipt); err != nil {
		return TerminalReceipt{}, err
	}
	return receipt, nil
}

func CanonicalTerminalReceipt(receipt TerminalReceipt) ([]byte, error) {
	if err := validateTerminalReceipt(receipt); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return nil, fmt.Errorf("encode terminal receipt: %w", err)
	}
	return raw, nil
}
