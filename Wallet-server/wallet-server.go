package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"golang.org/x/crypto/ripemd160"
)

type WalletServer struct {
	port    uint16
	gateway string
}

func NewWalletServer(port uint16, gateway string) *WalletServer {
	return &WalletServer{
		port:    port,
		gateway: gateway,
	}
}

func (ws *WalletServer) Port() uint16 {
	return ws.port
}

func (ws *WalletServer) Gateway() string {
	return ws.gateway
}

// Wallet structure
type Wallet struct {
	privateKey        *ecdsa.PrivateKey
	publicKey         *ecdsa.PublicKey
	blockchainAddress string
}

// NewWallet creates a new wallet with ECDSA cryptographic keys
func NewWallet() *Wallet {
	w := new(Wallet)

	// Generate ECDSA private key using P256 elliptic curve
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Printf("ERROR: Failed to generate private key: %v", err)
		return nil
	}
	w.privateKey = privateKey
	w.publicKey = &w.privateKey.PublicKey

	// Generate blockchain address from public key
	// Step 1: SHA256 hash of public key
	h2 := sha256.New()
	h2.Write(w.publicKey.X.Bytes())
	h2.Write(w.publicKey.Y.Bytes())
	digest2 := h2.Sum(nil)

	// Step 2: RIPEMD160 hash
	h3 := ripemd160.New()
	h3.Write(digest2)
	digest3 := h3.Sum(nil)

	// Step 3: Convert to hex string
	w.blockchainAddress = hex.EncodeToString(digest3)

	return w
}

// PrivateKeyStr returns the private key as a hex string
func (w *Wallet) PrivateKeyStr() string {
	return hex.EncodeToString(w.privateKey.D.Bytes())
}

// PublicKeyStr returns the public key as a hex string
func (w *Wallet) PublicKeyStr() string {
	return hex.EncodeToString(append(w.publicKey.X.Bytes(), w.publicKey.Y.Bytes()...))
}

// BlockchainAddress returns the blockchain address
func (w *Wallet) BlockchainAddress() string {
	return w.blockchainAddress
}

// Wallet HTTP handler - creates a new wallet
func (ws *WalletServer) Wallet(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		w.Header().Add("Content-Type", "application/json")

		// Create new wallet
		wallet := NewWallet()
		if wallet == nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Failed to create wallet",
			})
			return
		}

		// Prepare response (include frontend-friendly field names)
		response := struct {
			Address            string `json:"address"`
			PrivateKey         string `json:"privateKey"`
			PublicKey          string `json:"publicKey"`
			PrivateKeyLegacy   string `json:"private_key"`
			PublicKeyLegacy    string `json:"public_key"`
			BlockchainAddress  string `json:"blockchain_address"`
		}{
			Address:           wallet.BlockchainAddress(),
			PrivateKey:        wallet.PrivateKeyStr(),
			PublicKey:         wallet.PublicKeyStr(),
			PrivateKeyLegacy:  wallet.PrivateKeyStr(),
			PublicKeyLegacy:   wallet.PublicKeyStr(),
			BlockchainAddress: wallet.BlockchainAddress(),
		}

		log.Printf("New wallet created: %s", wallet.BlockchainAddress())

		// Send response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)

	default:
		log.Printf("ERROR: Invalid HTTP Method for /wallet")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid HTTP method",
		})
	}
}

// Index serves the main HTML page
func (ws *WalletServer) Index(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Use filepath.Join for cross-platform compatibility
		templatePath := filepath.Join("templates", "index.html")

		t, err := template.ParseFiles(templatePath)
		if err != nil {
			log.Printf("ERROR: Failed to parse template: %v", err)
			log.Printf("Looking for template at: %s", templatePath)
			http.Error(w, "Internal Server Error - Template not found", http.StatusInternalServerError)
			return
		}

		err = t.Execute(w, nil)
		if err != nil {
			log.Printf("ERROR: Failed to execute template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

// proxyToGateway forwards the request to the blockchain server
func (ws *WalletServer) proxyToGateway(path string, w http.ResponseWriter, r *http.Request) {
	url := ws.gateway + path
	var body io.Reader
	if r.Body != nil {
		body = r.Body
	}

	req, err := http.NewRequest(r.Method, url, body)
	if err != nil {
		log.Printf("ERROR: Failed to create proxy request: %v", err)
		http.Error(w, "Gateway error", http.StatusBadGateway)
		return
	}
	req.Header = r.Header.Clone()
	req.Header.Del("Host")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("ERROR: Proxy request failed: %v", err)
		http.Error(w, "Blockchain server unreachable. Is it running on "+ws.gateway+"?", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// proxyTransactions handles /transactions - transforms POST body for blockchain API
func (ws *WalletServer) proxyTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			SenderAddress    string  `json:"senderAddress"`
			RecipientAddress string  `json:"recipientAddress"`
			Value            float32 `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
			return
		}
		body := map[string]interface{}{
			"sender":    req.SenderAddress,
			"recipient": req.RecipientAddress,
			"value":     req.Value,
		}
		bodyBytes, _ := json.Marshal(body)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		r.ContentLength = int64(len(bodyBytes))
	}
	ws.proxyToGateway("/transactions", w, r)
}

// Run starts the wallet server
func (ws *WalletServer) Run() {
	http.HandleFunc("/", ws.Index)
	http.HandleFunc("/wallet", ws.Wallet)
	http.HandleFunc("/blockchain", func(w http.ResponseWriter, r *http.Request) {
		ws.proxyToGateway("/", w, r)
	})
	http.HandleFunc("/transactions", ws.proxyTransactions)
	http.HandleFunc("/mine", func(w http.ResponseWriter, r *http.Request) {
		ws.proxyToGateway("/mine", w, r)
	})
	// Balance / cryptocurrency total for an address (proxies to blockchain)
	http.HandleFunc("/amount", func(w http.ResponseWriter, r *http.Request) {
		path := "/amount"
		if q := r.URL.RawQuery; q != "" {
			path += "?" + q
		}
		ws.proxyToGateway(path, w, r)
	})

	log.Printf("Starting Wallet Server on port %d", ws.Port())
	log.Printf("Blockchain Gateway: %s", ws.Gateway())
	log.Printf("Open http://localhost:%d in your browser", ws.Port())
	log.Fatal(http.ListenAndServe("0.0.0.0:"+strconv.Itoa(int(ws.port)), nil))
}
