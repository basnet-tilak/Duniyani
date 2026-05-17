package core

import (
	"crypto/sha256"
)

// MerkleTree represents a Merkle tree.
type MerkleTree struct {
	RootNode *MerkleNode
}

// MerkleNode represents a node in the Merkle tree.
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

// NewMerkleNode creates a new Merkle tree node.
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	node := MerkleNode{}

	if left == nil && right == nil {
		// This is a leaf node, so we hash the data.
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	} else {
		// This is an internal node, so we concatenate and hash the data of children.
		var prevHashes []byte
		if left != nil {
			prevHashes = append(prevHashes, left.Data...)
		}
		if right != nil {
			prevHashes = append(prevHashes, right.Data...)
		}
		hash := sha256.Sum256(prevHashes)
		node.Data = hash[:]
	}

	node.Left = left
	node.Right = right

	return &node
}

// NewMerkleTree creates a Merkle tree from a slice of data (transaction hashes).
func NewMerkleTree(data [][]byte) *MerkleTree {
	if len(data) == 0 {
		// If there are no transactions, the Merkle root is the hash of an empty set.
		hash := sha256.Sum256([]byte{})
		root := &MerkleNode{Data: hash[:]}
		return &MerkleTree{RootNode: root}
	}

	var nodes []MerkleNode
	// Create leaf nodes
	for _, datum := range data {
		node := NewMerkleNode(nil, nil, datum)
		nodes = append(nodes, *node)
	}

	// Build the tree level by level
	for len(nodes) > 1 {
		// If there is an odd number of nodes, duplicate the last one
		if len(nodes)%2 != 0 {
			nodes = append(nodes, nodes[len(nodes)-1])
		}

		var newLevel []MerkleNode
		for i := 0; i < len(nodes); i += 2 {
			node := NewMerkleNode(&nodes[i], &nodes[i+1], nil)
			newLevel = append(newLevel, *node)
		}
		nodes = newLevel
	}

	// The last remaining node is the root
	tree := MerkleTree{&nodes[0]}

	return &tree
}
