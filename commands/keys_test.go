package commands

import (
	"crypto/rand"
	"testing"
)

func TestBech32Roundtrip(t *testing.T) {
	data := make([]byte, 113)
	rand.Read(data)

	packed, err := bech32ishPack("HRP", data)
	if err != nil {
		t.Error(err)
	}

	unpacked, err := bech32ishUnpack(packed)
	if err != nil {
		t.Error(err)
	}

	if len(data) != len(unpacked) {
		t.Fatal("length mismatch")
	}

	for i, v := range data {
		if v != unpacked[i] {
			t.Fatal("data mismatch")
		}
	}
}
