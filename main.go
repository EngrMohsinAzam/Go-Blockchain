package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
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

func (b *Block) Print() {
	fmt.Printf("timestamp       %d\n", b.timestamp)
	fmt.Printf("nonce           %d\n", b.nonce)
	fmt.Printf("previous_hash   %s\n", b.previousHash)
	for _, tx := range b.transaction {
		tx.Print()
	}
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
	bc.CreateBlock(nonce, previousHash)
	log.Println("action=mining, status=success")
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

// VerifyTransaction checks if the sender has sufficient balance and valid signature
func (bc *Blockchain) VerifyTransaction(sender string, value float32,
	senderPublicKey *ecdsa.PublicKey, signature *Signature) bool {
	// Mining rewards don't need verification
	if sender == MINING_SENDER {
		return true
	}

	// Verify signature if provided
	if senderPublicKey != nil && signature != nil {
		// Create a transaction hash for verification
		tx := NewTransaction(sender, "", value, nil) // recipient doesn't matter for verification
		if !bc.VerifyTransactionSignature(senderPublicKey, signature, tx) {
			log.Printf("action=verify_transaction, status=failed, reason=invalid_signature\n")
			return false
		}
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

// VerifyTransactionSignature verifies the ECDSA signature of a transaction
func (bc *Blockchain) VerifyTransactionSignature(publicKey *ecdsa.PublicKey,
	signature *Signature, tx *Transaction) bool {
	m, _ := json.Marshal(tx)
	h := sha256.Sum256([]byte(m))
	return ecdsa.Verify(publicKey, h[:], signature.R, signature.S)
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

func (bc *Blockchain) Print() {
	for i, block := range bc.chain {
		fmt.Printf("%s Chain %d %s\n", strings.Repeat("=", 25), i, strings.Repeat("=", 25))
		block.Print()
	}
	fmt.Printf("%s\n", strings.Repeat("*", 25))
}

// AddTransaction adds a transaction to the pool after verification
func (bc *Blockchain) AddTransaction(sender string, recipient string, value float32,
	signature *Signature) bool {
	// For non-mining transactions, signature is required
	if sender != MINING_SENDER && signature == nil {
		log.Printf("action=add_transaction, status=failed, reason=missing_signature\n")
		return false
	}

	tx := NewTransaction(sender, recipient, value, signature)

	// Verify the transaction (balance check, signature if applicable)
	if !bc.VerifyTransaction(sender, value, nil, signature) {
		log.Printf("action=add_transaction, status=failed, reason=verification_failed\n")
		return false
	}

	bc.transactionPool = append(bc.transactionPool, tx)
	log.Printf("action=add_transaction, status=success, sender=%s, recipient=%s, value=%.2f\n",
		sender, recipient, value)
	return true
}

// AddTransactionWithPublicKey adds a transaction with public key verification
func (bc *Blockchain) AddTransactionWithPublicKey(sender string, recipient string,
	value float32, senderPublicKey *ecdsa.PublicKey, signature *Signature) bool {

	if sender != MINING_SENDER && (senderPublicKey == nil || signature == nil) {
		log.Printf("action=add_transaction, status=failed, reason=missing_credentials\n")
		return false
	}

	tx := NewTransaction(sender, recipient, value, signature)

	// Verify the transaction with public key
	if !bc.VerifyTransaction(sender, value, senderPublicKey, signature) {
		log.Printf("action=add_transaction, status=failed, reason=verification_failed\n")
		return false
	}

	bc.transactionPool = append(bc.transactionPool, tx)
	log.Printf("action=add_transaction, status=success, sender=%s, recipient=%s, value=%.2f\n",
		sender, recipient, value)
	return true
}

type Transaction struct {
	sender    string
	recipient string
	value     float32
	signature *Signature
}

// Signature represents an ECDSA signature
type Signature struct {
	R *big.Int
	S *big.Int
}

func (bc *Blockchain) copyTransactionPool() []*Transaction {
	transactions := make([]*Transaction, 0)
	for _, tx := range bc.transactionPool {
		transactions = append(transactions, NewTransaction(
			tx.sender,
			tx.recipient,
			tx.value,
			tx.signature))
	}
	return transactions
}

func NewTransaction(sender string, recipient string, value float32, signature *Signature) *Transaction {
	return &Transaction{
		sender:    sender,
		recipient: recipient,
		value:     value,
		signature: signature,
	}
}

func (t *Transaction) Print() {
	fmt.Printf("Sender: %s, Recipient: %s, Value: %.2f", t.sender, t.recipient, t.value)
	if t.signature != nil {
		fmt.Printf(", Signed: Yes")
	}
	fmt.Println()
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

func init() {
	log.SetPrefix("Blockchain: ")
}

func main() {
	myBlockchainAddress := "my_blockchain_address_123"
	blockchain := NewBlockchain(myBlockchainAddress)
	blockchain.Print()

	fmt.Println("\n--- Adding transactions to pool ---")
	blockchain.AddTransaction("Alice", "Bob", 50.0, nil)
	blockchain.AddTransaction("Bob", "Charlie", 25.0, nil)

	time.Sleep(10 * time.Millisecond)
	blockchain.Mining()
	blockchain.Print()

	fmt.Println("\n--- Adding more transactions ---")
	blockchain.AddTransaction("Charlie", "David", 15.0, nil)

	time.Sleep(10 * time.Millisecond)
	blockchain.Mining()
	blockchain.Print()

	fmt.Println("\n--- Testing verification (should fail) ---")
	blockchain.AddTransaction("Alice", "Bob", 1000.0, nil) // Alice doesn't have 1000

	fmt.Printf("\nmy balance: %.2f\n", blockchain.calculateTotalAmount("my_blockchain_address_123"))
	fmt.Printf("Charlie balance: %.2f\n", blockchain.calculateTotalAmount("Charlie"))
}
