package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Transaction represents a blockchain transaction.
type Transaction struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Data      string `json:"data"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

// Block represents a single block in the chain.
type Block struct {
	Index        uint64        `json:"index"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []*Transaction `json:"transactions"`
	PrevHash     string        `json:"prev_hash"`
	Hash         string        `json:"hash"`
	Validator    string        `json:"validator"`
	Nonce        uint64        `json:"nonce"`
}

// computeHash computes SHA-256 hash of the block.
func computeHash(b *Block) string {
	data := fmt.Sprintf("%d%d%s%s%s%d", b.Index, b.Timestamp, marshalTx(b.Transactions), b.PrevHash, b.Validator, b.Nonce)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

func marshalTx(txs []*Transaction) string {
	b, _ := json.Marshal(txs)
	return string(b)
}

// MerkleTree computes a Merkle root from transactions.
type MerkleTree struct {
	Root   *MerkleNode
	Leaves []*MerkleNode
}

// MerkleNode is a node in the Merkle tree.
type MerkleNode struct {
	Hash  string
	Left  *MerkleNode
	Right *MerkleNode
}

// NewMerkleTree builds a Merkle tree from transactions.
func NewMerkleTree(txs []*Transaction) *MerkleTree {
	if len(txs) == 0 {
		return &MerkleTree{Root: &MerkleNode{Hash: sha256Hash("")}}
	}
	var leaves []*MerkleNode
	for _, tx := range txs {
		leaves = append(leaves, &MerkleNode{Hash: sha256Hash(tx.ID)})
	}
	root := buildTree(leaves)
	return &MerkleTree{Root: root, Leaves: leaves}
}

func buildTree(nodes []*MerkleNode) *MerkleNode {
	if len(nodes) == 1 {
		return nodes[0]
	}
	var nextLevel []*MerkleNode
	for i := 0; i < len(nodes); i += 2 {
		if i+1 < len(nodes) {
			combined := sha256Hash(nodes[i].Hash + nodes[i+1].Hash)
			nextLevel = append(nextLevel, &MerkleNode{Hash: combined, Left: nodes[i], Right: nodes[i+1]})
		} else {
			combined := sha256Hash(nodes[i].Hash + nodes[i].Hash)
			nextLevel = append(nextLevel, &MerkleNode{Hash: combined, Left: nodes[i]})
		}
	}
	return buildTree(nextLevel)
}

func sha256Hash(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// StateDB is a simple key-value state database.
type StateDB struct {
	mu    sync.RWMutex
	store map[string]string
}

// NewStateDB creates a new state database.
func NewStateDB() *StateDB {
	return &StateDB{store: make(map[string]string)}
}

// Get retrieves a value.
func (db *StateDB) Get(key string) (string, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	v, ok := db.store[key]
	return v, ok
}

// Set stores a value.
func (db *StateDB) Set(key, value string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.store[key] = value
}

// Delete removes a key.
func (db *StateDB) Delete(key string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	delete(db.store, key)
}

// Blockchain manages the chain of blocks.
type Blockchain struct {
	mu          sync.RWMutex
	chain       []*Block
	pendingTx   []*Transaction
	stateDB     *StateDB
	difficulty  int
	onBlockMined func(*Block)
}

// NewBlockchain creates a new blockchain.
func NewBlockchain(difficulty int) *Blockchain {
	bc := &Blockchain{
		stateDB:    NewStateDB(),
		difficulty: difficulty,
	}
	bc.chain = append(bc.chain, bc.createGenesisBlock())
	return bc
}

// SetOnBlockMined sets the callback for when a block is mined.
func (bc *Blockchain) SetOnBlockMined(fn func(*Block)) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.onBlockMined = fn
}

func (bc *Blockchain) createGenesisBlock() *Block {
	return &Block{
		Index:     0,
		Timestamp: time.Now().Unix(),
		PrevHash:  "0",
		Hash:      sha256Hash("genesis"),
		Validator: "system",
	}
}

// AddTransaction adds a transaction to the pending pool.
func (bc *Blockchain) AddTransaction(tx *Transaction) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.pendingTx = append(bc.pendingTx, tx)
}

// MintBlock creates a new block with PoA consensus.
func (bc *Blockchain) MintBlock(validator string) (*Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if len(bc.chain) == 0 {
		return nil, fmt.Errorf("chain is empty")
	}

	lastBlock := bc.chain[len(bc.chain)-1]
	newBlock := &Block{
		Index:        lastBlock.Index + 1,
		Timestamp:    time.Now().Unix(),
		Transactions: bc.pendingTx,
		PrevHash:     lastBlock.Hash,
		Validator:    validator,
	}

	// PoA: simple proof-of-authority with difficulty prefix
	for {
		newBlock.Nonce++
		newBlock.Hash = computeHash(newBlock)
		if newBlock.Hash[:bc.difficulty] == repeatZeros(bc.difficulty) {
			break
		}
	}

	bc.chain = append(bc.chain, newBlock)
	bc.pendingTx = nil

	// Store task results in state DB
	for _, tx := range newBlock.Transactions {
		bc.stateDB.Set("tx:"+tx.ID, tx.Data)
	}

	if bc.onBlockMined != nil {
		go bc.onBlockMined(newBlock)
	}

	return newBlock, nil
}

// GetChain returns the full chain.
func (bc *Blockchain) GetChain() []*Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	result := make([]*Block, len(bc.chain))
	copy(result, bc.chain)
	return result
}

// GetStateDB returns the state database.
func (bc *Blockchain) GetStateDB() *StateDB {
	return bc.stateDB
}

// PendingCount returns the number of pending transactions.
func (bc *Blockchain) PendingCount() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.pendingTx)
}

func repeatZeros(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "0"
	}
	return s
}
