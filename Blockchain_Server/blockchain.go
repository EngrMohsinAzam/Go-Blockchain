package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"
	"time"
)

const (
	MiningDifficulty = 3
	MINING_SENDER    = "THE BLOCKCHAIN"
	MINING_REWARD    = 1.0
)

type Block struct {
	timestamp    int64
	nonce        int
	previousHash string
	transaction  []*Transaction
}

func NewBlock(nonce int, previousHash string, transaction []*Transaction) *Block {
	b := new(Block)
	b.timestamp = time.Now().UnixNano()
	b.nonce = nonce
	b.previousHash = previousHash
	b.transaction = transaction
	return b
}

func (b *Block) Hash() string {
	m, _ := json.Marshal(b)
	hash := sha256.Sum256([]byte(m))
	return hex.EncodeToString(hash[:])
}

func (b *Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Timestamp    int64          `json:"timestamp"`
		Nonce        int            `json:"nonce"`
		PreviousHash string         `json:"previous_hash"`
		Transaction  []*Transaction `json:"transaction"`
	}{
		Timestamp:    b.timestamp,
		Nonce:        b.nonce,
		PreviousHash: b.previousHash,
		Transaction:  b.transaction,
	})
}

type Blockchain struct {
	transactionPool   []*Transaction
	chain             []*Block
	blockchainAddress string
}

func NewBlockchain(blockchainAddress string) *Blockchain {
	bc := new(Blockchain)
	bc.transactionPool = []*Transaction{}
	bc.blockchainAddress = blockchainAddress
	genesisHash := strings.Repeat("0", 64)
	bc.CreateBlock(0, genesisHash)
	return bc
}

func (bc *Blockchain) CreateBlock(nonce int, previousHash string) *Block {
	b := NewBlock(nonce, previousHash, bc.transactionPool)
	bc.chain = append(bc.chain, b)
	bc.transactionPool = []*Transaction{} // Clear transaction pool
	return b
}

func (bc *Blockchain) Mining() bool {
	bc.AddTransaction(MINING_SENDER, bc.blockchainAddress, MINING_REWARD, nil)
	nonce := bc.ProofOfWork()
	previousHash := bc.LastBlock().Hash()
	b := bc.CreateBlock(nonce, previousHash)
	log.Printf(">>> BLOCK MINED | %d transactions in block | block hash: %s...\n", len(b.transaction), b.Hash()[:16])
	for i, tx := range b.transaction {
		log.Printf("    [%d] %s -> %s : %.2f\n", i+1, tx.sender, tx.recipient, tx.value)
	}
	return true
}

func (bc *Blockchain) calculateTotalAmount(address string) float32 {
	total := float32(0)
	for _, block := range bc.chain {
		for _, tx := range block.transaction {
			if tx.recipient == address {
				total += tx.value
			}
			if tx.sender == address {
				total -= tx.value
			}
		}
	}
	return total
}

func (bc *Blockchain) VerifyTransaction(sender string, value float32) bool {
	// Mining rewards don't need verification
	if sender == MINING_SENDER {
		return true
	}

	// Calculate sender's current balance
	balance := bc.calculateTotalAmount(sender)

	// Check if sender has enough balance for the transaction
	if balance < value {
		log.Printf("action=verify_transaction, status=failed, sender=%s, balance=%.2f, required=%.2f\n",
			sender, balance, value)
		return false
	}

	log.Printf("action=verify_transaction, status=success, sender=%s, balance=%.2f, amount=%.2f\n",
		sender, balance, value)
	return true
}

func (bc *Blockchain) ProofOfWork() int {
	transactions := bc.copyTransactionPool()
	previousHash := bc.LastBlock().Hash()
	nonce := 0
	for !bc.ValidProof(transactions, previousHash, nonce, MiningDifficulty) {
		nonce++
	}
	return nonce
}

func (bc *Blockchain) ValidProof(transactions []*Transaction, previousHash string, nonce int, difficulty int) bool {
	zeros := strings.Repeat("0", difficulty)
	guessBlock := Block{
		timestamp:    0,
		nonce:        nonce,
		previousHash: previousHash,
		transaction:  transactions,
	}
	guessHash := guessBlock.Hash()
	return guessHash[:difficulty] == zeros
}

func (bc *Blockchain) LastBlock() *Block {
	return bc.chain[len(bc.chain)-1]
}

// GetAllTransactions returns all transactions from the chain (mined blocks)
func (bc *Blockchain) GetAllTransactions() []*Transaction {
	var all []*Transaction
	for _, block := range bc.chain {
		all = append(all, block.transaction...)
	}
	return all
}

// GetTransactionsByAddress returns transactions involving the given address
func (bc *Blockchain) GetTransactionsByAddress(address string) []*Transaction {
	var result []*Transaction
	for _, tx := range bc.GetAllTransactions() {
		if tx.sender == address || tx.recipient == address {
			result = append(result, tx)
		}
	}
	return result
}

// AddTransaction adds a transaction to the pool after verification
// signature parameter is included for API compatibility but not used in this implementation
func (bc *Blockchain) AddTransaction(sender string, recipient string, value float32, signature interface{}) bool {
	// Verify the transaction
	if !bc.VerifyTransaction(sender, value) {
		log.Printf("action=add_transaction, status=failed, reason=insufficient_balance\n")
		return false
	}

	tx := NewTransaction(sender, recipient, value)
	bc.transactionPool = append(bc.transactionPool, tx)
	log.Printf(">>> TX ADDED TO POOL | %s -> %s : %.2f (pending until mined)\n", sender, recipient, value)
	return true
}

func (bc *Blockchain) copyTransactionPool() []*Transaction {
	transactions := make([]*Transaction, 0)
	for _, tx := range bc.transactionPool {
		transactions = append(transactions, NewTransaction(
			tx.sender,
			tx.recipient,
			tx.value))
	}
	return transactions
}

type Transaction struct {
	sender    string
	recipient string
	value     float32
}

func NewTransaction(sender string, recipient string, value float32) *Transaction {
	return &Transaction{sender: sender, recipient: recipient, value: value}
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sender    string  `json:"sender"`
		Recipient string  `json:"recipient"`
		Value     float32 `json:"value"`
	}{
		Sender:    t.sender,
		Recipient: t.recipient,
		Value:     t.value,
	})
}
