# Blockchain API - Learning Guide

A simple REST API for your Go blockchain. Run the server and test with **curl** or **browser**.

---

## Start the Server

```bash
cd Blockchain_Server
go run .
```

Server runs on **http://localhost:5000**

---

## The Transaction Flow (How Blockchain Works)

```
1. MINE first      → Miner gets coins (reward)
2. CREATE tx       → Add to pool (pending)
3. MINE again      → Pool goes into a block, confirmed
4. VIEW history    → See all transactions
```

---

## API Endpoints

### 1. Mine a Block (Get Coins)

**GET** `http://localhost:5000/mine`

- Creates a new block and gives the miner a reward
- Run this first so the miner has balance to send

```bash
curl http://localhost:5000/mine
```

---

### 2. Create a Transaction

**POST** `http://localhost:5000/api/transactions`

Body (JSON):
```json
{
  "sender": "miner_blockchain_address_123",
  "recipient": "your_wallet_address",
  "value": 1.5
}
```

```bash
curl -X POST http://localhost:5000/api/transactions \
  -H "Content-Type: application/json" \
  -d "{\"sender\":\"miner_blockchain_address_123\",\"recipient\":\"alice123\",\"value\":1.5}"
```

**Note:** Sender must have enough balance. The miner gets coins from mining.

---

### 3. Get Pending Transactions (Pool)

**GET** `http://localhost:5000/api/transactions`

- Returns transactions waiting to be mined

```bash
curl http://localhost:5000/api/transactions
```

---

### 4. Get Transaction History (All Mined)

**GET** `http://localhost:5000/api/transactions/history`

- Returns all transactions in the chain

```bash
curl http://localhost:5000/api/transactions/history
```

---

### 5. Get Balance

**GET** `http://localhost:5000/amount?blockchain_address=miner_blockchain_address_123`

```bash
curl "http://localhost:5000/amount?blockchain_address=miner_blockchain_address_123"
```

---

### 6. Get Transactions by Address

**GET** `http://localhost:5000/api/transactions/address?address=miner_blockchain_address_123`

```bash
curl "http://localhost:5000/api/transactions/address?address=miner_blockchain_address_123"
```

---

## Step-by-Step Test (Copy & Run)

### Terminal 1: Start server
```bash
cd D:\Projects\Go-Blockchain\Blockchain_Server
go run .
```

### Terminal 2: Run these curls in order

**Step 1 – Mine (miner gets 1 coin)**
```bash
curl http://localhost:5000/mine
```

**Step 2 – Check miner balance**
```bash
curl "http://localhost:5000/amount?blockchain_address=miner_blockchain_address_123"
```

**Step 3 – Create a transaction (miner → alice)**
```bash
curl -X POST http://localhost:5000/api/transactions -H "Content-Type: application/json" -d "{\"sender\":\"miner_blockchain_address_123\",\"recipient\":\"alice123\",\"value\":0.5}"
```

**Step 4 – See pending transactions**
```bash
curl http://localhost:5000/api/transactions
```

**Step 5 – Mine again (confirm the transaction)**
```bash
curl http://localhost:5000/mine
```

**Step 6 – See full history**
```bash
curl http://localhost:5000/api/transactions/history
```

**Step 7 – See alice's transactions**
```bash
curl "http://localhost:5000/api/transactions/address?address=alice123"
```

---

## Quick Reference

| Action        | Method | URL                                      |
|---------------|--------|------------------------------------------|
| Mine block    | GET    | /mine                                    |
| Create tx     | POST   | /api/transactions                        |
| Pending pool  | GET    | /api/transactions                        |
| Full history  | GET    | /api/transactions/history                |
| By address    | GET    | /api/transactions/address?address=xxx    |
| Balance       | GET    | /amount?blockchain_address=xxx           |
| Blockchain    | GET    | /                                        |

---

## Miner Address

Default miner address: `miner_blockchain_address_123`

This address receives mining rewards. Use it as `sender` to send coins to other addresses.

---

## Wallet Server – Balance (Cryptocurrency Total)

When the **Wallet server** is running (e.g. `go run .` in `Wallet-server`, port 8080), it proxies balance requests to the blockchain:

**GET** `http://localhost:8080/amount?blockchain_address=YOUR_ADDRESS`

Returns JSON: `{"blockchain_address":"...", "amount": 1.5}`

The wallet UI shows **Your Cryptocurrency Total** at the top and calls this API when you create a wallet, send, get test coins, or mine. Use **Refresh Balance** to update.
