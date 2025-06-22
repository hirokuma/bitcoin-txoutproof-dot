package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/awalterschulze/gographviz"
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

// MerkleProofData defines the structure for the data following the block header,
// resembling parts of a MerkleBlock message (BIP 37).
type MerkleProofData struct {
	TotalTransactions uint32
	HashNum           uint64 // Number of hashes that follow
	Hashes            [][32]byte
	VBitsNum          uint64 // Number of bytes in vBits (flags)
	VBits             []bool // Flag, unpacked field
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

// doubleSha256 computes SHA256(SHA256(data))
func doubleSha256(data []byte) [32]byte {
	hash1 := sha256.Sum256(data)
	hash2 := sha256.Sum256(hash1[:])
	return hash2
}

// readVarInt reads a Bitcoin-style variable-length integer (CompactSize).
// Reference: https://developer.bitcoin.org/reference/transactions.html#compactsize-unsigned-integers
func readVarInt(r io.Reader) (uint64, error) {
	var discriminant uint8
	err := binary.Read(r, binary.LittleEndian, &discriminant)
	if err != nil {
		return 0, err // Handles io.EOF if reader is empty or read fails
	}

	switch discriminant {
	case 0xfd:
		var val uint16
		err = binary.Read(r, binary.LittleEndian, &val)
		if err != nil {
			return 0, err
		}
		return uint64(val), nil
	case 0xfe:
		var val uint32
		err = binary.Read(r, binary.LittleEndian, &val)
		if err != nil {
			return 0, err
		}
		return uint64(val), nil
	case 0xff:
		var val uint64
		err = binary.Read(r, binary.LittleEndian, &val)
		if err != nil {
			return 0, err
		}
		return val, nil
	default:
		return uint64(discriminant), nil
	}
}

func nodeLabel(hash []byte) string {
	// Returns a label for the node, formatted as a hex string.
	// The hash is reversed to match the common Bitcoin representation.
	return fmt.Sprintf("\"%.8s...\"", hex.EncodeToString(reverseBytes(hash[:])))
}

func decodeTxOutProofData(decodedBytes []byte, blockHeader *BlockHeader, proofData *MerkleProofData) error {
	var err error

	// 5. Check if decodedBytes has enough data for a block header (80 bytes)
	if len(decodedBytes) < 80 {
		return fmt.Errorf("decoded data is less than 80 bytes (got %d bytes). Cannot parse block header", len(decodedBytes))
	}

	// 6. Take the first 80 bytes for the header
	headerBytes := decodedBytes[:80]

	// 7. Decode the block header
	reader := bytes.NewReader(headerBytes)
	err = binary.Read(reader, binary.LittleEndian, blockHeader)
	if err != nil {
		return fmt.Errorf("failed to decode block header: %w", err)
	}

	// 8. Print the decoded block header
	fmt.Printf("//Decoded Block Header (first 80 bytes):\n")
	fmt.Printf("//  Version:         %d (0x%x)\n", blockHeader.Version, blockHeader.Version)
	// PrevBlockHash and MerkleRoot are typically displayed with bytes reversed from their in-memory representation.
	fmt.Printf("//  Prev Block Hash: %s\n", hex.EncodeToString(reverseBytes(blockHeader.PrevBlockHash[:])))
	fmt.Printf("//  Merkle Root:     %s\n", hex.EncodeToString(reverseBytes(blockHeader.MerkleRoot[:])))
	fmt.Printf("//  Timestamp:       %d (%s UTC)\n", blockHeader.Timestamp, time.Unix(int64(blockHeader.Timestamp), 0).UTC().Format(time.RFC3339))
	fmt.Printf("//  Bits (Target):   %d (0x%x)\n", blockHeader.Bits, blockHeader.Bits)
	fmt.Printf("//  Nonce:           %d (0x%x)\n", blockHeader.Nonce, blockHeader.Nonce)

	// 9. Process data after the block header (Merkle proof like data)
	remainingBytes := decodedBytes[80:]
	if len(remainingBytes) == 0 {
		return fmt.Errorf("no additional data found after block header")
	}

	merkleReader := bytes.NewReader(remainingBytes)

	// Read TotalTransactions
	if err = binary.Read(merkleReader, binary.LittleEndian, &proofData.TotalTransactions); err != nil {
		return fmt.Errorf("reading TotalTransactions from Merkle proof data: %w", err)
	}

	// Read HashNum (hash_count)
	proofData.HashNum, err = readVarInt(merkleReader)
	if err != nil {
		return fmt.Errorf("reading HashNum (hash_count) from Merkle proof data: %w", err)
	}

	// Read Hashes
	proofData.Hashes = make([][32]byte, proofData.HashNum)
	for i := uint64(0); i < proofData.HashNum; i++ {
		if _, err = io.ReadFull(merkleReader, proofData.Hashes[i][:]); err != nil {
			return fmt.Errorf("reading hash #%d from Merkle proof data: %w", i+1, err)
		}
	}

	// Read VBitsNum (flag_count)
	proofData.VBitsNum, err = readVarInt(merkleReader)
	if err != nil {
		return fmt.Errorf("eading VBitsNum (flag_count) from Merkle proof data: %w", err)
	}

	// Read VBits (flags)
	var vBits []byte
	if proofData.VBitsNum > 0 {
		vBits = make([]byte, proofData.VBitsNum)
		if _, err = io.ReadFull(merkleReader, vBits); err != nil {
			return fmt.Errorf("reading VBits (flags) from Merkle proof data: %w", err)
		}

		for _, b := range vBits {
			// Iterate through each bit in the byte, from LSB (0) to MSB (7)
			for i := 0; i < 8; i++ {
				// Check if the i-th bit is set (1)
				isSet := (b>>uint(i))&1 == 1
				proofData.VBits = append(proofData.VBits, isSet)
			}
		}
	} else {
		vBits = []byte{}
		proofData.VBits = []bool{} // Ensure it's an empty slice, not nil
	}

	// 10. Print the decoded Merkle proof data
	fmt.Printf("//Decoded Merkle Proof Data (following header):\n")
	fmt.Printf("//  Total Transactions: %d\n", proofData.TotalTransactions)
	fmt.Printf("//  Hash Count (hash_num): %d\n", proofData.HashNum)
	fmt.Printf("//  Hashes:\n")
	for i, hash := range proofData.Hashes {
		fmt.Printf("//    %d: %s\n", i+1, hex.EncodeToString(reverseBytes(hash[:])))
	}
	fmt.Printf("//  Flag Bytes Count (vbits_num): %d\n", proofData.VBitsNum)
	fmt.Printf("//  Flag Bits (vBits): %s\n", hex.EncodeToString(vBits))

	return nil
}

// buildMerkleTreeDot constructs the Merkle tree and returns its Graphviz DOT representation.
func buildMerkleTreeDot(totalTx uint32, vbits []bool, hashes [][32]byte) (string, [32]byte, error) {
	graph := gographviz.NewGraph()
	if err := graph.SetName("G"); err != nil {
		return "", [32]byte{}, err
	}
	if err := graph.SetDir(true); err != nil { // Directed graph
		return "", [32]byte{}, err
	}

	// Calculate the total height of the tree
	height := 0
	for (1 << uint(height)) < int(totalTx) {
		height++
	}

	merkleBranch := MerkleBranch{}
	root, err := merkleBranch.CreateMerkleBranch(vbits, hashes, height)
	if err != nil {
		return "", [32]byte{}, fmt.Errorf("failed to create Merkle branch: %w", err)
	}
	g, err := merkleBranch.BuildGraph()
	if err != nil {
		return "", [32]byte{}, fmt.Errorf("failed to build Merkle branch graph: %w", err)
	}

	return g, root, nil
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

	var blockHeader BlockHeader
	var proofData MerkleProofData

	err = decodeTxOutProofData(decodedBytes, &blockHeader, &proofData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to decode transaction output proof data: %v\n", err)
		os.Exit(1)
	}

	// 13. Build Merkle Tree and generate Graphviz DOT output
	dotString, root, err := buildMerkleTreeDot(proofData.TotalTransactions, proofData.VBits, proofData.Hashes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building Merkle tree: %v\n", err)
		os.Exit(1)
	}
	if blockHeader.MerkleRoot != root {
		fmt.Fprintf(os.Stderr, "Merkle Root mismatch!!\n")
	}
	fmt.Println(dotString)
}
