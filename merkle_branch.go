package main

import (
	"encoding/hex"
	"fmt"

	"github.com/awalterschulze/gographviz"
)

type MerkleNode struct {
	Hash   []byte
	Parent *MerkleNode
	Left   *MerkleNode
	Right  *MerkleNode
}

func NewBinTree(parent *MerkleNode) *MerkleNode {
	return &MerkleNode{Parent: parent}
}

type MerkleBranch struct {
	binTree []*MerkleNode
	target  *MerkleNode
}

func (m *MerkleBranch) CreateMerkleBranch(vbits []bool, hashes [][32]byte, height int) ([32]byte, error) {
	m.binTree = []*MerkleNode{
		NewBinTree(nil), // Root node
	}

	var root [32]byte
	current := m.binTree[0]
	vbitsIndex := 0
	hashesIndex := 0
	depth := 0
	for {
		if hashesIndex == len(hashes) {
			if current.Left == nil || current.Left.Hash == nil || current.Right == nil || current.Right.Hash == nil {
				return [32]byte{}, fmt.Errorf("all hashes is parsed, but tree construction is not finished")
			}
			root = doubleSha256(append(current.Left.Hash, current.Right.Hash...))
			current.Hash = root[:]
			fmt.Printf("//Calculated Merkle Root: %x\n", reverseBytes(current.Hash))
			break
		}

		if current.Left != nil && current.Right != nil {
			if current.Hash == nil {
				hash := doubleSha256(append(current.Left.Hash, current.Right.Hash...))
				current.Hash = hash[:]
			}

			// go to up node
			current = current.Parent
			depth--
			continue
		}

		if current.Left != nil && current.Right == nil {
			// go to right node
			node := NewBinTree(current)
			m.binTree = append(m.binTree, node)
			current.Right = node

			current = node
			depth++
		}

		if !vbits[vbitsIndex] || depth == height {
			// set leaf information
			current.Hash = hashes[hashesIndex][:]
			hashesIndex++
			if vbits[vbitsIndex] && depth == height {
				m.target = current
			}

			current = current.Parent
			depth--
		} else {
			// add left node and go to it
			node := NewBinTree(current)
			m.binTree = append(m.binTree, node)
			current.Left = node

			current = node
			depth++
		}
		vbitsIndex++
		if vbitsIndex >= len(vbits) {
			return [32]byte{}, fmt.Errorf("vBits is too short, expected at least %d bits, got %d", len(m.binTree), vbitsIndex)
		}
	}

	return root, nil
}

func (m *MerkleBranch) BuildGraph() (string, error) {
	graph := gographviz.NewGraph()
	if err := graph.SetName("G"); err != nil {
		return "", err
	}
	if err := graph.SetDir(true); err != nil { // Directed graph
		return "", err
	}

	for _, node := range m.binTree {
		nodeID := fmt.Sprintf("node_%x", node.Hash[:])
		var label string
		if node.Parent == nil {
			label = fmt.Sprintf("\"%s\"", hex.EncodeToString(reverseBytes(node.Hash[:])))
		} else {
			label = nodeLabel(node.Hash)
		}
		var nodeShape string
		if node.Left == nil && node.Right == nil {
			nodeShape = "box" // Leaf nodes are boxes
		} else {
			nodeShape = "ellipse" // Non-leaf nodes are ellipses
		}
		var fillColor string
		if node.Left == nil && node.Right == nil {
			if node == m.target {
				fillColor = "lightcoral"
			} else {
				fillColor = "lightblue"
			}
		} else {
			fillColor = "white"
		}
		attrs := map[string]string{
			"label":     label,
			"shape":     nodeShape,
			"style":     "filled",
			"fillcolor": fillColor,
		}
		graph.AddNode("G", nodeID, attrs)

		if node.Left != nil && node.Right != nil {
			leftChildID := fmt.Sprintf("node_%x", node.Left.Hash[:])
			leftEdgeAttrs := map[string]string{"tailport": "sw"}
			rightChildID := fmt.Sprintf("node_%x", node.Right.Hash[:])
			rightEdgeAttrs := map[string]string{"tailport": "se"}
			graph.AddEdge(nodeID, leftChildID, true, leftEdgeAttrs)
			graph.AddEdge(nodeID, rightChildID, true, rightEdgeAttrs)
		}
	}

	return graph.String(), nil
}
