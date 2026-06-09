package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
)

type BlockchainServer struct {
	port       uint16
	blockchain *Blockchain
}

func NewBlockchainServer(port uint16, blockchain *Blockchain) *BlockchainServer {
	return &BlockchainServer{
		port:       port,
		blockchain: blockchain,
	}
}

func (bcs *BlockchainServer) Port() uint16 {
	return bcs.port
}

// CORS Middleware - enables cross-origin requests from wallet server
func (bcs *BlockchainServer) enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func (bcs *BlockchainServer) GetBlockchain(w http.ResponseWriter, r *http.Request) {
	bcs.enableCORS(w)

	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
		return

	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")

		// Create a response structure that includes the chain
		response := struct {
			Chain []*Block `json:"chain"`
		}{
			Chain: bcs.blockchain.chain,
		}

		jsonData, err := json.Marshal(response)
		if err != nil {
			log.Printf("ERROR: Failed to marshal blockchain: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Transactions(w http.ResponseWriter, r *http.Request) {
	bcs.enableCORS(w)

	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
		return

	case http.MethodGet:
		// Get transaction pool
		w.Header().Add("Content-Type", "application/json")
		type TransactionResponse struct {
			Transactions []*Transaction `json:"transactions"`
			Length       int            `json:"length"`
		}
		response := TransactionResponse{
			Transactions: bcs.blockchain.transactionPool,
			Length:       len(bcs.blockchain.transactionPool),
		}
		jsonData, err := json.Marshal(response)
		if err != nil {
			log.Printf("ERROR: Failed to marshal transactions: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)

	case http.MethodPost:
		// Add transaction
		w.Header().Add("Content-Type", "application/json")
		type AddTransactionRequest struct {
			Sender    string  `json:"sender"`
			Recipient string  `json:"recipient"`
			Value     float32 `json:"value"`
		}
		var req AddTransactionRequest
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&req)
		if err != nil {
			log.Printf("ERROR: Failed to decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error": "Invalid request body"}`)
			return
		}

		// Validate input
		if req.Sender == "" || req.Recipient == "" || req.Value <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error": "Invalid transaction data"}`)
			return
		}

		// Add transaction to pool
		success := bcs.blockchain.AddTransaction(req.Sender, req.Recipient, req.Value, nil)
		if success {
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, `{"message": "Transaction added successfully"}`)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error": "Failed to add transaction - insufficient balance"}`)
		}

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Mine(w http.ResponseWriter, r *http.Request) {
	bcs.enableCORS(w)

	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
		return

	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")

		// Perform mining (Mining() function adds reward transaction automatically)
		success := bcs.blockchain.Mining()
		if success {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"message": "Mining successful"}`)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, `{"error": "Mining failed"}`)
		}

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) StartMine(w http.ResponseWriter, r *http.Request) {
	bcs.enableCORS(w)

	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
		return

	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")

		// Start mining in background
		go func() {
			bcs.blockchain.Mining()
		}()
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"message": "Mining started in background"}`)

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Amount(w http.ResponseWriter, r *http.Request) {
	bcs.enableCORS(w)

	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
		return

	case http.MethodGet:
		// Get blockchain address from query parameter
		blockchainAddress := r.URL.Query().Get("blockchain_address")
		w.Header().Add("Content-Type", "application/json")

		if blockchainAddress == "" {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error": "blockchain_address parameter is required"}`)
			return
		}

		// Calculate amount for address
		amount := bcs.blockchain.calculateTotalAmount(blockchainAddress)
		type AmountResponse struct {
			BlockchainAddress string  `json:"blockchain_address"`
			Amount            float32 `json:"amount"`
		}
		response := AmountResponse{
			BlockchainAddress: blockchainAddress,
			Amount:            amount,
		}

		jsonData, err := json.Marshal(response)
		if err != nil {
			log.Printf("ERROR: Failed to marshal response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Run() {
	// Core blockchain endpoints
	http.HandleFunc("/", bcs.GetBlockchain)
	http.HandleFunc("/transactions", bcs.Transactions)
	http.HandleFunc("/mine", bcs.Mine)
	http.HandleFunc("/mine/start", bcs.StartMine)
	http.HandleFunc("/amount", bcs.Amount)

	// Transaction API (learning-friendly, same logic, clearer paths)
	http.HandleFunc("/api/transactions", bcs.handleTransactionAPI)
	http.HandleFunc("/api/transactions/history", bcs.GetTransactionHistory)
	http.HandleFunc("/api/transactions/address", bcs.GetTransactionsByAddress)

	log.Printf("Starting Blockchain Server on port %d", bcs.Port())
	log.Fatal(http.ListenAndServe("0.0.0.0:"+strconv.Itoa(int(bcs.Port())), nil))
}
