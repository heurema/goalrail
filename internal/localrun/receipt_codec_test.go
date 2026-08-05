package localrun

import (
	"bytes"
	"reflect"
	"testing"
)

func TestTerminalReceiptCodecRetainsExactValidatedCanonicalBytes(t *testing.T) {
	receipt := validFixtureReceipt()
	raw, err := CanonicalTerminalReceipt(receipt)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeTerminalReceipt(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, receipt) {
		t.Fatalf("decoded receipt differs: %+v", decoded)
	}
	if _, err := DecodeTerminalReceipt(bytes.NewReader(append(raw, []byte(` {}`)...))); err == nil {
		t.Fatal("terminal receipt codec accepted trailing JSON")
	}
}
