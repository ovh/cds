package uilive

import (
	"bytes"
	"fmt"
	"testing"
)

func TestWriter(t *testing.T) {
	w := New()
	b := &bytes.Buffer{}
	w.Out = b
	w.Start()
	for i := 0; i < 2; i++ {
		fmt.Fprintln(w, "foo")
	}
	w.Stop()
	want := "foo\nfoo\n"
	if b.String() != want {
		t.Fatalf("want %q, got %q", want, b.String())
	}
}
