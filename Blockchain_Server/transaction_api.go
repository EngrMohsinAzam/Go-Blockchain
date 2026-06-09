package main

import (
	"encoding/json"
	"io"
	"net/http"
)

// TransactionAPI provides clean REST endpoints for transactions (for learning)
// This file shows the flow: create tx -> pool -> mine -> chain

// TransactionRequest - JSON body for creating a transaction
type TransactionRequest struct {
	Sender    string  `json:"sender"`
	Recipient string  `json:"recipient"`
	Value     float32 `json:"value"`
}

// TransactionResponse - JSON response for a single transaction
type TransactionResponse struct {
	Sender    string  `json:"sender"`
	Recipient string  `json:"recipient"`
	Value     float32 `json:"value"`
}

// PoolResponse - GET /api/transactions (pending pool)
type PoolResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	Count        int                   `json:"count"`
	Message      string                `json:"message"`
}

// HistoryResponse - GET /api/transactions/history (all mined)
type HistoryResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	Count        int                   `json:"count"`
}

// CreateResponse - POST /api/transactions (create)
type CreateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// toResponse converts internal Transaction to API format
func txToResponse(tx *Transaction) TransactionResponse {
	return TransactionResponse{
		Sender:    tx.sender,
		Recipient: tx.recipient,
		Value:     tx.value,
	}
}

func txListToResponse(txs []*Transaction) []TransactionResponse {
	out := make([]TransactionResponse, len(txs))
	for i, tx := range txs {
		out[i] = txToResponse(tx)
	}
	return out
}

// handleTransactionAPI routes GET -> pool, POST -> create
func (bcs *BlockchainServer) handleTransactionAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		bcs.GetTransactionPool(w, r)
	case http.MethodPost:
		bcs.CreateTransaction(w, r)
	case http.MethodOptions:
		bcs.enableCORS(w)
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// GetTransactionPool returns pending transactions (waiting to be mined)
func (bcs *BlockchainServer) GetTransactionPool(w http.ResponseWriter, r *http.Request) {
	bcs.enableCORS(w)

	txs := bcs.blockchain.transactionPool
	resp := PoolResponse{
		Transactions: txListToResponse(txs),
		Count:        len(txs),
		Message:      "These transactions are pending. Call POST /mine to add them to the chain.",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CreateTransaction adds a new transaction to the pool
func (bcs *BlockchainServer) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	bcs.enableCORS(w)

	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CreateResponse{Success: false, Message: "Invalid JSON body"})
		return
	}

	if req.Sender == "" || req.Recipient == "" || req.Value <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CreateResponse{Success: false, Message: "sender, recipient, and value (>0) required"})
		return
	}

	ok := bcs.blockchain.AddTransaction(req.Sender, req.Recipient, req.Value, nil)
	w.Header().Set("Content-Type", "application/json")
	if ok {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateResponse{Success: true, Message: "Transaction added to pool. Mine a block to confirm."})
	} else {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CreateResponse{Success: false, Message: "Transaction failed - sender has insufficient balance"})
	}
}

// GetTransactionHistory returns all transactions from mined blocks
func (bcs *BlockchainServer) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	bcs.enableCORS(w)

	txs := bcs.blockchain.GetAllTransactions()
	resp := HistoryResponse{
		Transactions: txListToResponse(txs),
		Count:        len(txs),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetTransactionsByAddress returns transactions for a specific address
func (bcs *BlockchainServer) GetTransactionsByAddress(w http.ResponseWriter, r *http.Request) {
	bcs.enableCORS(w)

	address := r.URL.Query().Get("address")
	if address == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "query param 'address' required"}`)
		return
	}

	txs := bcs.blockchain.GetTransactionsByAddress(address)
	resp := HistoryResponse{
		Transactions: txListToResponse(txs),
		Count:        len(txs),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
