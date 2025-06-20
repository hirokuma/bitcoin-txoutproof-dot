package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

// BlockHeader defines the structure of a Bitcoin block header.
// Reference: https://developer.bitcoin.org/reference/block_chain.html#block-headers
type BlockHeader struct {
	Version       int32
	PrevBlockHash [32]byte
	MerkleRoot    [32]byte
	Timestamp     uint32
	Bits          uint32
	Nonce         uint32
}

// reverseBytes reverses a byte slice. Useful for displaying Bitcoin hashes
// in the commonly seen big-endian format, as they are often stored little-endian.
func reverseBytes(data []byte) []byte {
	reversed := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		reversed[i] = data[len(data)-1-i]
	}
	return reversed
}

func main() {
	// 1. コマンドライン引数の数をチェック
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Error: Please provide exactly one hexadecimal string argument.\n")
		fmt.Fprintf(os.Stderr, "Usage: %s <hex_string>\n", os.Args[0])
		os.Exit(1)
	}

	// 2. 第一引数を取得
	hexInput := os.Args[1]

	// 3. 16進数文字列を []uint8 に変換
	//    DecodeStringは、入力文字列の長さが奇数の場合や、
	//    16進数として不正な文字が含まれている場合にエラーを返します。
	decodedBytes, err := hex.DecodeString(hexInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to convert hexadecimal string: %v\n", err)
		os.Exit(1)
	}

	// 4. 変換できた []uint8 を再度16進数文字列に変換して標準出力
	//    This step confirms the input was correctly processed.
	//    The original hexInput and hexOutput should be the same (ignoring case differences).
	hexOutput := hex.EncodeToString(decodedBytes)
	fmt.Println("Re-encoded full hex string:", hexOutput)

	// 5. Check if decodedBytes has enough data for a block header (80 bytes)
	if len(decodedBytes) < 80 {
		fmt.Fprintf(os.Stderr, "Error: Decoded data is less than 80 bytes (got %d bytes). Cannot parse block header.\n", len(decodedBytes))
		os.Exit(1)
	}

	// 6. Take the first 80 bytes for the header
	headerBytes := decodedBytes[:80]

	// 7. Decode the block header
	reader := bytes.NewReader(headerBytes)
	var blockHeader BlockHeader
	err = binary.Read(reader, binary.LittleEndian, &blockHeader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to decode block header: %v\n", err)
		os.Exit(1)
	}

	// 8. Print the decoded block header
	fmt.Println("\nDecoded Block Header (first 80 bytes):")
	fmt.Printf("  Version:         %d (0x%x)\n", blockHeader.Version, blockHeader.Version)
	// PrevBlockHash and MerkleRoot are typically displayed with bytes reversed from their in-memory representation.
	fmt.Printf("  Prev Block Hash: %s\n", hex.EncodeToString(reverseBytes(blockHeader.PrevBlockHash[:])))
	fmt.Printf("  Merkle Root:     %s\n", hex.EncodeToString(reverseBytes(blockHeader.MerkleRoot[:])))
	fmt.Printf("  Timestamp:       %d (%s UTC)\n", blockHeader.Timestamp, time.Unix(int64(blockHeader.Timestamp), 0).UTC().Format(time.RFC3339))
	fmt.Printf("  Bits (Target):   %d (0x%x)\n", blockHeader.Bits, blockHeader.Bits)
	fmt.Printf("  Nonce:           %d (0x%x)\n", blockHeader.Nonce, blockHeader.Nonce)
}
