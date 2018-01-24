package compressutil

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"testing"
)

func TestCompressUtil_CompressDecompress(t *testing.T) {
	input := map[string]interface{}{
		"sample":       "data",
		"verification": "process",
	}

	// Encode input into JSON
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(input); err != nil {
		t.Fatal(err)
	}

	inputJSONBytes := buf.Bytes()
	// Test nil configuration
	if _, err := Compress(inputJSONBytes, nil); err == nil {
		t.Fatal("expected an error")
	}

	// Test invalid configuration
	if _, err := Compress(inputJSONBytes, &CompressionConfig{}); err == nil {
		t.Fatal("expected an error")
	}

	// Compress input using lzw format
	compressedJSONBytes, err := Compress(inputJSONBytes, &CompressionConfig{
		Type: CompressionTypeLzw,
	})
	if err != nil {
		t.Fatal("expected an error")
	}
	if len(compressedJSONBytes) == 0 {
		t.Fatal("failed to compress data in lzw format")
	}
	// Check the presense of the canary
	if compressedJSONBytes[0] != CompressionCanaryLzw {
		t.Fatal("bad: compression canary: expected: %d actual: %d", CompressionCanaryLzw, compressedJSONBytes[0])
	}

	// Decompress the input and check the output
	decompressedJSONBytes, uncompressed, err := Decompress(compressedJSONBytes)
	if err != nil {
		t.Fatal(err)
	}
	if uncompressed {
		t.Fatal("failed to recognize compressed data")
	}
	if len(decompressedJSONBytes) == 0 {
		t.Fatal("failed to decompress lzw formatted data")
	}

	if string(inputJSONBytes) != string(decompressedJSONBytes) {
		t.Fatalf("bad: mismatch: inputJSONBytes: %s\n decompressedJSONBytes: %s", string(inputJSONBytes), string(decompressedJSONBytes))
	}

	// Compress input using Gzip format, assume DefaultCompression
	compressedJSONBytes, err = Compress(inputJSONBytes, &CompressionConfig{
		Type: CompressionTypeGzip,
	})
	if err != nil {
		t.Fatal("expected an error")
	}
	if len(compressedJSONBytes) == 0 {
		t.Fatal("failed to compress data in lzw format")
	}
	// Check the presense of the canary
	if compressedJSONBytes[0] != CompressionCanaryGzip {
		t.Fatal("bad: compression canary: expected: %d actual: %d", CompressionCanaryGzip, compressedJSONBytes[0])
	}

	// Decompress the input and check the output
	decompressedJSONBytes, uncompressed, err = Decompress(compressedJSONBytes)
	if err != nil {
		t.Fatal(err)
	}
	if uncompressed {
		t.Fatal("failed to recognize compressed data")
	}
	if len(decompressedJSONBytes) == 0 {
		t.Fatal("failed to decompress lzw formatted data")
	}

	if string(inputJSONBytes) != string(decompressedJSONBytes) {
		t.Fatalf("bad: mismatch: inputJSONBytes: %s\n decompressedJSONBytes: %s", string(inputJSONBytes), string(decompressedJSONBytes))
	}

	// Compress input using Gzip format: DefaultCompression
	compressedJSONBytes, err = Compress(inputJSONBytes, &CompressionConfig{
		Type:                 CompressionTypeGzip,
		GzipCompressionLevel: gzip.DefaultCompression,
	})
	if err != nil {
		t.Fatal("expected an error")
	}
	if len(compressedJSONBytes) == 0 {
		t.Fatal("failed to compress data in lzw format")
	}
	// Check the presense of the canary
	if compressedJSONBytes[0] != CompressionCanaryGzip {
		t.Fatal("bad: compression canary: expected: %d actual: %d", CompressionCanaryGzip, compressedJSONBytes[0])
	}

	// Decompress the input and check the output
	decompressedJSONBytes, uncompressed, err = Decompress(compressedJSONBytes)
	if err != nil {
		t.Fatal(err)
	}
	if uncompressed {
		t.Fatal("failed to recognize compressed data")
	}
	if len(decompressedJSONBytes) == 0 {
		t.Fatal("failed to decompress lzw formatted data")
	}

	if string(inputJSONBytes) != string(decompressedJSONBytes) {
		t.Fatalf("bad: mismatch: inputJSONBytes: %s\n decompressedJSONBytes: %s", string(inputJSONBytes), string(decompressedJSONBytes))
	}

	// Compress input using Gzip format, BestCompression
	compressedJSONBytes, err = Compress(inputJSONBytes, &CompressionConfig{
		Type:                 CompressionTypeGzip,
		GzipCompressionLevel: gzip.BestCompression,
	})
	if err != nil {
		t.Fatal("expected an error")
	}
	if len(compressedJSONBytes) == 0 {
		t.Fatal("failed to compress data in lzw format")
	}
	// Check the presense of the canary
	if compressedJSONBytes[0] != CompressionCanaryGzip {
		t.Fatal("bad: compression canary: expected: %d actual: %d", CompressionCanaryGzip, compressedJSONBytes[0])
	}

	// Decompress the input and check the output
	decompressedJSONBytes, uncompressed, err = Decompress(compressedJSONBytes)
	if err != nil {
		t.Fatal(err)
	}
	if uncompressed {
		t.Fatal("failed to recognize compressed data")
	}
	if len(decompressedJSONBytes) == 0 {
		t.Fatal("failed to decompress lzw formatted data")
	}

	if string(inputJSONBytes) != string(decompressedJSONBytes) {
		t.Fatalf("bad: mismatch: inputJSONBytes: %s\n decompressedJSONBytes: %s", string(inputJSONBytes), string(decompressedJSONBytes))
	}

	// Compress input using Gzip format, BestSpeed
	compressedJSONBytes, err = Compress(inputJSONBytes, &CompressionConfig{
		Type:                 CompressionTypeGzip,
		GzipCompressionLevel: gzip.BestSpeed,
	})
	if err != nil {
		t.Fatal("expected an error")
	}
	if len(compressedJSONBytes) == 0 {
		t.Fatal("failed to compress data in lzw format")
	}
	// Check the presense of the canary
	if compressedJSONBytes[0] != CompressionCanaryGzip {
		t.Fatal("bad: compression canary: expected: %d actual: %d",
			CompressionCanaryGzip, compressedJSONBytes[0])
	}

	// Decompress the input and check the output
	decompressedJSONBytes, uncompressed, err = Decompress(compressedJSONBytes)
	if err != nil {
		t.Fatal(err)
	}
	if uncompressed {
		t.Fatal("failed to recognize compressed data")
	}
	if len(decompressedJSONBytes) == 0 {
		t.Fatal("failed to decompress lzw formatted data")
	}

	if string(inputJSONBytes) != string(decompressedJSONBytes) {
		t.Fatalf("bad: mismatch: inputJSONBytes: %s\n decompressedJSONBytes: %s", string(inputJSONBytes), string(decompressedJSONBytes))
	}
}
