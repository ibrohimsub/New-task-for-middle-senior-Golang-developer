package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID         string  `json:"id"`
	Balance    float64 `json:"balance"`
	Identified bool    `json:"identified"`
}

type Transaction struct {
	ID         string    `json:"id"`
	WalletID   string    `json:"walletId"`
	Amount     float64   `json:"amount"`
	Timestamp  time.Time `json:"timestamp"`
}

var (
	transactions = struct {
		sync.RWMutex
		data map[string][]Transaction
	}{
		data: make(map[string][]Transaction),
	}

	wallets = struct {
		sync.RWMutex
		data map[string]Wallet
	}{
		data: make(map[string]Wallet),
	}

	maxBalanceIdentified   = 100000.0
	maxBalanceUnidentified = 10000.0
	secretKey              = "your-secret-key"
)

const (
	transactionDateFormat = "2006-01-02T15:04:05Z07:00"
)

func checkWalletExistsHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-UserId")
	digest := r.Header.Get("X-Digest")

	if validateDigest(r, digest) {
		wallets.RLock()
		defer wallets.RUnlock()

		wallet, found := wallets.data[userId]
		if found {
			walletJSON, err := json.Marshal(wallet)
			if err != nil {
				http.Error(w, "Failed to serialize wallet", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(walletJSON)
		} else {
			http.Error(w, "Wallet not found", http.StatusNotFound)
		}
	} else {
		http.Error(w, "Invalid digest", http.StatusUnauthorized)
	}
}

func depositToWalletHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-UserId")
	digest := r.Header.Get("X-Digest")

	if validateDigest(r, digest) {
		wallets.Lock()
		defer wallets.Unlock()

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		type DepositRequest struct {
			Amount float64 `json:"amount"`
		}

		var req DepositRequest
		err = json.Unmarshal(body, &req)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		wallet, found := wallets.data[userId]
		if !found {
			http.Error(w, "Wallet not found", http.StatusNotFound)
			return
		}

		if wallet.Identified && wallet.Balance+req.Amount > maxBalanceIdentified {
			http.Error(w, "Exceeded maximum balance for identified wallets", http.StatusBadRequest)
			return
		}

		if !wallet.Identified && wallet.Balance+req.Amount > maxBalanceUnidentified {
			http.Error(w, "Exceeded maximum balance for unidentified wallets", http.StatusBadRequest)
			return
		}

		wallet.Balance += req.Amount
		wallets.data[userId] = wallet

		// Add transaction to the transactions map
		transaction := Transaction{
			ID:        uuid.New().String(),
			WalletID:  userId,
			Amount:    req.Amount,
			Timestamp: time.Now(),
		}
		transactions.Lock()
		transactions.data[userId] = append(transactions.data[userId], transaction)
		transactions.Unlock()

		walletJSON, err := json.Marshal(wallet)
		if err != nil {
			http.Error(w, "Failed to serialize wallet", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(walletJSON)
	} else {
		http.Error(w, "Invalid digest", http.StatusUnauthorized)
	}
}

func getMonthlyOperationsHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-UserId")
	digest := r.Header.Get("X-Digest")

	if validateDigest(r, digest) {
		transactions.RLock()
		userTransactions := transactions.data[userId]
		transactions.RUnlock()

		// Calculate total count and sum of deposits for the current month
		currentMonth := time.Now().Month()
		totalCount := 0
		totalSum := 0.0

		for _, transaction := range userTransactions {
			if transaction.Timestamp.Month() == currentMonth {
				totalCount++
				totalSum += transaction.Amount
			}
		}

		// Prepare the response JSON
		response := struct {
			Count int     `json:"count"`
			Sum   float64 `json:"sum"`
		}{
			Count: totalCount,
			Sum:   totalSum,
		}

		responseJSON, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseJSON)
	} else {
		http.Error(w, "Invalid digest", http.StatusUnauthorized)
	}
}

func getWalletBalanceHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-UserId")
	digest := r.Header.Get("X-Digest")

	if validateDigest(r, digest) {
		wallets.RLock()
		defer wallets.RUnlock()

		wallet, found := wallets.data[userId]
		if found {
			balanceJSON, err := json.Marshal(map[string]float64{"balance": wallet.Balance})
			if err != nil {
				http.Error(w, "Failed to serialize balance", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(balanceJSON)
		} else {
			http.Error(w, "Wallet not found", http.StatusNotFound)
		}
	} else {
		http.Error(w, "Invalid digest", http.StatusUnauthorized)
	}
}

func createWalletHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-UserId")
	digest := r.Header.Get("X-Digest")

	if validateDigest(r, digest) {
		wallets.Lock()
		defer wallets.Unlock()

		_, found := wallets.data[userId]
		if found {
			http.Error(w, "Wallet already exists", http.StatusBadRequest)
			return
		}

		// Create a new wallet with a balance of 0
		wallet := Wallet{
			ID:         userId,
			Balance:    0.0,
			Identified: false,
		}
		wallets.data[userId] = wallet

		walletJSON, err := json.Marshal(wallet)
		if err != nil {
			http.Error(w, "Failed to serialize wallet", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(walletJSON)
	} else {
		http.Error(w, "Invalid digest", http.StatusUnauthorized)
	}
}

func validateDigest(r *http.Request, digest string) bool {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		return false
	}
	r.Body = ioutil.NopCloser(bytes.NewReader(body)) // Reset the request body

	h := hmac.New(sha1.New, []byte(secretKey))
	h.Write(body)
	expectedDigest := hex.EncodeToString(h.Sum(nil))

	return expectedDigest == digest
}

func setupRoutes() {
	http.HandleFunc("/wallets/check", checkWalletExistsHandler)
	http.HandleFunc("/wallets/deposit", depositToWalletHandler)
	http.HandleFunc("/wallets/operations", getMonthlyOperationsHandler)
	http.HandleFunc("/wallets/balance", getWalletBalanceHandler)
	http.HandleFunc("/wallets/create", createWalletHandler)
}

func startServer() {
	log.Println("Starting server on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func main() {
	setupRoutes()
	startServer()
}
