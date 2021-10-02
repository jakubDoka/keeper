package khtml

import "testing"

func Benchmark(b *testing.B) {
	var html Html
	var dump []byte

	for i := 0; i < b.N; i++ {
		dump = html.Tag("html").Text("Hello!").Close(dump[:0])
	}
}
