package zcashblob

import "testing"

func TestBlake2bPersonalChunkSplits(t *testing.T) {
	var personalization [16]byte
	copy(personalization[:], "ChunkSplitTest__")
	message := make([]byte, 3*128+1)
	for i := range message {
		message[i] = byte(i % 251)
	}
	want := blake2bPersonal(personalization, message)
	for split := 0; split <= len(message); split++ {
		got := blake2bPersonal(personalization, message[:split], nil, message[split:])
		if got != want {
			t.Fatalf("split %d: got %x, want %x", split, got, want)
		}
	}
}
