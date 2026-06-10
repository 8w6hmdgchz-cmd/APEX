package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/nousresearch/agent-os-v2/pkg/common"
)

// ---------------------------------------------------------------------------
// Secure Merkle Tree — [32]byte SHA-256 based
// ---------------------------------------------------------------------------

// SecureMerkleNode is a node in the secure Merkle tree using fixed-size hashes.
type SecureMerkleNode struct {
	Hash  [32]byte
	Left  *SecureMerkleNode
	Right *SecureMerkleNode
}

// SecureMerkleTree provides cryptographic verification of transactions within a block.
type SecureMerkleTree struct {
	Root   *SecureMerkleNode
	Leaves []*SecureMerkleNode
}

// hashTransaction produces a [32]byte SHA-256 hash of a transaction's canonical form.
func hashTransaction(tx *Transaction) [32]byte {
	data, _ := json.Marshal(tx)
	return sha256.Sum256(data)
}

// hashPair concatenates two [32]byte hashes and returns the SHA-256 of the result.
func hashPair(left, right [32]byte) [32]byte {
	var buf [64]byte
	copy(buf[:32], left[:])
	copy(buf[32:], right[:])
	return sha256.Sum256(buf[:])
}

// NewSecureMerkleTree builds a Merkle tree from a slice of transactions.
// If txs is empty the root hash is sha256 of the empty string.
func NewSecureMerkleTree(txs []*Transaction) *SecureMerkleTree {
	logger := common.DefaultLogger()

	if len(txs) == 0 {
		root := sha256.Sum256([]byte{})
		node := &SecureMerkleNode{Hash: root}
		logger.Debug("SecureMerkleTree: created empty tree, root=%s", hex.EncodeToString(root[:]))
		return &SecureMerkleTree{Root: node, Leaves: nil}
	}

	// Build leaf level
	leaves := make([]*SecureMerkleNode, len(txs))
	for i, tx := range txs {
		h := hashTransaction(tx)
		leaves[i] = &SecureMerkleNode{Hash: h}
	}

	root := buildSecureTree(leaves)
	logger.Debug("SecureMerkleTree: built tree with %d leaves, root=%s", len(leaves), hex.EncodeToString(root.Hash[:]))
	return &SecureMerkleTree{Root: root, Leaves: leaves}
}

// buildSecureTree recursively pairs nodes upward until a single root remains.
func buildSecureTree(nodes []*SecureMerkleNode) *SecureMerkleNode {
	if len(nodes) == 1 {
		return nodes[0]
	}
	var nextLevel []*SecureMerkleNode
	for i := 0; i < len(nodes); i += 2 {
		if i+1 < len(nodes) {
			parent := &SecureMerkleNode{
				Hash:  hashPair(nodes[i].Hash, nodes[i+1].Hash),
				Left:  nodes[i],
				Right: nodes[i+1],
			}
			nextLevel = append(nextLevel, parent)
		} else {
			// Duplicate the last node when the level has an odd count
			parent := &SecureMerkleNode{
				Hash:  hashPair(nodes[i].Hash, nodes[i].Hash),
				Left:  nodes[i],
				Right: nodes[i],
			}
			nextLevel = append(nextLevel, parent)
		}
	}
	return buildSecureTree(nextLevel)
}

// MerkleRoot returns the root hash of the tree.
func (mt *SecureMerkleTree) MerkleRoot() [32]byte {
	if mt.Root == nil {
		return [32]byte{}
	}
	return mt.Root.Hash
}

// Verify checks whether a transaction belongs to this Merkle tree.
func (mt *SecureMerkleTree) Verify(tx *Transaction) bool {
	txHash := hashTransaction(tx)
	proof, index := mt.GenerateProof(tx)
	if proof == nil {
		return false
	}
	return VerifyProof(txHash, proof, mt.Root.Hash, index)
}

// GenerateProof produces a Merkle inclusion proof for the given transaction.
// Returns the sibling-hashes along the path from the leaf to the root and the
// leaf index. Returns (nil, -1) when the transaction is not found.
func (mt *SecureMerkleTree) GenerateProof(tx *Transaction) ([][32]byte, int) {
	txHash := hashTransaction(tx)
	leafIndex := -1
	for i, leaf := range mt.Leaves {
		if leaf.Hash == txHash {
			leafIndex = i
			break
		}
	}
	if leafIndex < 0 {
		return nil, -1
	}

	var proof [][32]byte
	nodes := mt.Leaves
	idx := leafIndex

	for len(nodes) > 1 {
		var nextLevel []*SecureMerkleNode
		for i := 0; i < len(nodes); i += 2 {
			if i+1 < len(nodes) {
				// Record sibling
				if i == idx {
					proof = append(proof, nodes[i+1].Hash)
				} else if i+1 == idx {
					proof = append(proof, nodes[i].Hash)
				}
				parent := &SecureMerkleNode{
					Hash:  hashPair(nodes[i].Hash, nodes[i+1].Hash),
					Left:  nodes[i],
					Right: nodes[i+1],
				}
				nextLevel = append(nextLevel, parent)
			} else {
				if i == idx {
					proof = append(proof, nodes[i].Hash)
				}
				parent := &SecureMerkleNode{
					Hash:  hashPair(nodes[i].Hash, nodes[i].Hash),
					Left:  nodes[i],
					Right: nodes[i],
				}
				nextLevel = append(nextLevel, parent)
			}
		}
		// Update idx to parent position
		idx = idx / 2
		nodes = nextLevel
	}

	return proof, leafIndex
}

// VerifyProof verifies a Merkle inclusion proof.
//   - leafHash:  the [32]byte hash of the leaf transaction
//   - proof:     sibling hashes along the path leaf → root
//   - root:      the expected Merkle root
//   - leafIndex: 0-based index of the leaf
func VerifyProof(leafHash [32]byte, proof [][32]byte, root [32]byte, leafIndex int) bool {
	current := leafHash
	idx := leafIndex
	for _, sibling := range proof {
		if idx%2 == 0 {
			current = hashPair(current, sibling)
		} else {
			current = hashPair(sibling, current)
		}
		idx = idx / 2
	}
	return current == root
}

// ---------------------------------------------------------------------------
// Cross-Chain Bridge
// ---------------------------------------------------------------------------

// ChainID identifies a blockchain network.
type ChainID int

const (
	ChainLocal     ChainID = iota // Local agent-os chain
	ChainEthereum                 // Ethereum mainnet / testnet
	ChainAPEX                     // APEX sidechain
)

// String returns a human-readable name for the chain.
func (c ChainID) String() string {
	switch c {
	case ChainLocal:
		return "Local"
	case ChainEthereum:
		return "Ethereum"
	case ChainAPEX:
		return "APEX"
	default:
		return fmt.Sprintf("Chain(%d)", int(c))
	}
}

// BridgeStatus represents the lifecycle state of a cross-chain transfer.
type BridgeStatus int

const (
	BridgeStatusPending   BridgeStatus = iota // Submitted, waiting for confirmation
	BridgeStatusConfirmed                     // Source-chain proof verified
	BridgeStatusCompleted                     // Asset delivered on destination chain
	BridgeStatusFailed                        // Permanent failure
)

// String returns a human-readable label for the status.
func (s BridgeStatus) String() string {
	switch s {
	case BridgeStatusPending:
		return "Pending"
	case BridgeStatusConfirmed:
		return "Confirmed"
	case BridgeStatusCompleted:
		return "Completed"
	case BridgeStatusFailed:
		return "Failed"
	default:
		return fmt.Sprintf("Status(%d)", int(s))
	}
}

// CrossChainTx represents a single cross-chain asset transfer.
type CrossChainTx struct {
	SrcChain ChainID     `json:"src_chain"`
	DstChain ChainID     `json:"dst_chain"`
	TxID     string      `json:"tx_id"`
	Amount   float64     `json:"amount"`
	Status   BridgeStatus `json:"status"`
}

// CrossChainBridge manages pending and completed cross-chain transfers.
type CrossChainBridge struct {
	mu        sync.RWMutex
	pending   map[string]*CrossChainTx
	completed map[string]*CrossChainTx
	logger    *common.Logger
}

// NewCrossChainBridge creates a new bridge instance.
func NewCrossChainBridge() *CrossChainBridge {
	return &CrossChainBridge{
		pending:   make(map[string]*CrossChainTx),
		completed: make(map[string]*CrossChainTx),
		logger:    common.DefaultLogger(),
	}
}

var (
	ErrBridgeNilTx       = errors.New("bridge: nil transaction")
	ErrBridgeEmptyTxID   = errors.New("bridge: empty transaction ID")
	ErrBridgeDuplicate   = errors.New("bridge: duplicate transaction ID")
	ErrBridgeNotFound    = errors.New("bridge: transaction not found")
	ErrBridgeNotPending  = errors.New("bridge: transaction is not in pending state")
	ErrBridgeInvalidProof = errors.New("bridge: invalid proof")
)

// Submit registers a new cross-chain transfer request.
func (b *CrossChainBridge) Submit(tx *CrossChainTx) error {
	if tx == nil {
		return ErrBridgeNilTx
	}
	if tx.TxID == "" {
		return ErrBridgeEmptyTxID
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.pending[tx.TxID]; exists {
		return fmt.Errorf("%w: %s", ErrBridgeDuplicate, tx.TxID)
	}
	if _, exists := b.completed[tx.TxID]; exists {
		return fmt.Errorf("%w: %s", ErrBridgeDuplicate, tx.TxID)
	}

	tx.Status = BridgeStatusPending
	b.pending[tx.TxID] = tx
	b.logger.Info("Bridge: submitted cross-chain tx %s (%s → %s, amount=%.4f)",
		tx.TxID, tx.SrcChain, tx.DstChain, tx.Amount)
	return nil
}

// Confirm verifies a proof for a pending cross-chain transfer and moves it to
// the confirmed state. The proof is expected to be a Merkle inclusion proof
// serialised as raw bytes; for this implementation any non-empty proof is
// accepted as valid (real validation would verify against the source chain's
// state root).
func (b *CrossChainBridge) Confirm(txID string, proof []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	tx, exists := b.pending[txID]
	if !exists {
		// Might already be completed
		if _, done := b.completed[txID]; done {
			return fmt.Errorf("%w: already completed", ErrBridgeNotPending)
		}
		return fmt.Errorf("%w: %s", ErrBridgeNotFound, txID)
	}

	if tx.Status != BridgeStatusPending {
		return fmt.Errorf("%w: current status=%s", ErrBridgeNotPending, tx.Status)
	}

	if len(proof) == 0 {
		return ErrBridgeInvalidProof
	}

	tx.Status = BridgeStatusConfirmed
	b.logger.Info("Bridge: confirmed cross-chain tx %s", txID)

	// Move to completed (simulate finality)
	tx.Status = BridgeStatusCompleted
	delete(b.pending, txID)
	b.completed[txID] = tx
	b.logger.Info("Bridge: completed cross-chain tx %s", txID)

	return nil
}

// GetStatus returns the current status of a cross-chain transfer.
// Returns BridgeStatusPending (-1 sentinel is not used; returns the actual
// status) or an error-compatible status if not found.
func (b *CrossChainBridge) GetStatus(txID string) BridgeStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if tx, ok := b.pending[txID]; ok {
		return tx.Status
	}
	if tx, ok := b.completed[txID]; ok {
		return tx.Status
	}
	return BridgeStatusFailed // not-found treated as failed
}

// GetTransaction returns a copy of the cross-chain transaction if it exists.
func (b *CrossChainBridge) GetTransaction(txID string) (*CrossChainTx, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if tx, ok := b.pending[txID]; ok {
		cp := *tx
		return &cp, true
	}
	if tx, ok := b.completed[txID]; ok {
		cp := *tx
		return &cp, true
	}
	return nil, false
}

// PendingCount returns the number of pending cross-chain transfers.
func (b *CrossChainBridge) PendingCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.pending)
}

// CompletedCount returns the number of completed cross-chain transfers.
func (b *CrossChainBridge) CompletedCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.completed)
}

// ---------------------------------------------------------------------------
// Transaction Receipt — includes Merkle inclusion proof
// ---------------------------------------------------------------------------

// TransactionReceipt provides cryptographic proof that a transaction was
// included in a specific block.
type TransactionReceipt struct {
	TxID       string     `json:"tx_id"`
	BlockIndex int        `json:"block_index"`
	MerkleProof [][32]byte `json:"merkle_proof"`
	Index      int        `json:"index"` // leaf index inside the Merkle tree
	Root       [32]byte   `json:"root"`  // Merkle root of the block
}

// GetReceipt constructs a TransactionReceipt for the given transaction ID.
// It scans the chain, locates the block and transaction, builds a secure
// Merkle tree for that block, and generates an inclusion proof.
func (bc *Blockchain) GetReceipt(txID string) (*TransactionReceipt, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	logger := common.DefaultLogger()

	for _, block := range bc.chain {
		for _, tx := range block.Transactions {
			if tx.ID == txID {
				// Build secure Merkle tree for this block
				tree := NewSecureMerkleTree(block.Transactions)
				proof, index := tree.GenerateProof(tx)
				if proof == nil {
					logger.Error("GetReceipt: failed to generate proof for tx %s in block %d", txID, block.Index)
					return nil, fmt.Errorf("receipt: proof generation failed for tx %s", txID)
				}

				receipt := &TransactionReceipt{
					TxID:        txID,
					BlockIndex:  int(block.Index),
					MerkleProof: proof,
					Index:       index,
					Root:        tree.MerkleRoot(),
				}
				logger.Info("GetReceipt: generated receipt for tx %s (block=%d, proof_len=%d)",
					txID, block.Index, len(proof))
				return receipt, nil
			}
		}
	}

	logger.Warn("GetReceipt: tx %s not found in chain", txID)
	return nil, fmt.Errorf("receipt: transaction %s not found", txID)
}

// VerifyReceipt checks that a receipt's Merkle proof is valid against the
// provided root. This is a standalone function so external callers can verify
// receipts without access to the blockchain.
func VerifyReceipt(receipt *TransactionReceipt, tx *Transaction) bool {
	if receipt == nil || tx == nil {
		return false
	}
	txHash := hashTransaction(tx)
	return VerifyProof(txHash, receipt.MerkleProof, receipt.Root, receipt.Index)
}

// ComputeBlockMerkleRoot is a helper that returns the [32]byte Merkle root for
// a set of transactions. Useful when constructing enhanced blocks externally.
func ComputeBlockMerkleRoot(txs []*Transaction) [32]byte {
	tree := NewSecureMerkleTree(txs)
	return tree.MerkleRoot()
}

// EnhancedBlock extends the base Block with a computed MerkleRoot.
// Because the original Block struct lives in blockchain.go and we must not
// modify it, this wrapper carries the additional field.
type EnhancedBlock struct {
	*Block
	MerkleRoot [32]byte `json:"merkle_root"`
}

// NewEnhancedBlock wraps a Block and computes its MerkleRoot.
func NewEnhancedBlock(b *Block) *EnhancedBlock {
	return &EnhancedBlock{
		Block:      b,
		MerkleRoot: ComputeBlockMerkleRoot(b.Transactions),
	}
}
