package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID         string
	Balance    float64
	Identified bool
}

type Transaction struct {
	ID         string
	WalletID   string
	Amount     float64
	Timestamp  time.Time
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
		data: map[string]Wallet{
			"1": {ID: "1", Balance: 10000, Identified: false},
			"2": {ID: "2", Balance: 100000, Identified: true},
		},
	}

	maxBalanceIdentified   = 100000.0
	maxBalanceUnidentified = 10000.0
	secretKey              = "your-secret-key"
)

const (
	transactionDateFormat = "2006-01-02T15:04:05Z07:00"
)

func main() {
	http.HandleFunc("/wallets/check", checkWalletExistsHandler)
	http.HandleFunc("/wallets/deposit", depositToWalletHandler)
	http.HandleFunc("/wallets/operations", getMonthlyOperationsHandler)
	http.HandleFunc("/wallets/balance", getWalletBalanceHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func checkWalletExistsHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-UserId")
	digest := r.Header.Get("X-Digest")

	if validateDigest(r, digest) {
		wallets.RLock()
		defer wallets.RUnlock()

		wallet, found := wallets.data[userId]
		if found {
			walletJSON, _ := json.Marshal(wallet)
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

		var depositAmount float64
		err = json.Unmarshal(body, &depositAmount)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		wallet, found := wallets.data[userId]
		if !found {
			http.Error(w, "Wallet not found", http.StatusNotFound)
			return
		}

		if wallet.Identified && wallet.Balance+depositAmount > maxBalanceIdentified {
			http.Error(w, "Exceeded maximum balance for identified wallets", http.StatusBadRequest)
			return
		}

		if !wallet.Identified && wallet.Balance+depositAmount > maxBalanceUnidentified {
			http.Error(w, "Exceeded maximum balance for unidentified wallets", http.StatusBadRequest)
			return
		}

		wallet.Balance += depositAmount
		wallets.data[userId] = wallet

		// Add transaction to the transactions map
		transaction := Transaction{
			ID:        uuid.New().String(),
			WalletID:  userId,
			Amount:    depositAmount,
			Timestamp: time.Now(),
		}
		transactions.Lock()
		transactions.data[userId] = append(transactions.data[userId], transaction)
		transactions.Unlock()

		walletJSON, _ := json.Marshal(wallet)
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
			balanceJSON, _ := json.Marshal(map[string]float64{"balance": wallet.Balance})
			w.Header().Set("Content-Type", "application/json")
			w.Write(balanceJSON)
		} else {
			http.Error(w, "Wallet not found", http.StatusNotFound)
		}
	} else {
		http.Error(w, "Invalid digest", http.StatusUnauthorized)
	}
}

func validateDigest(r *http.Request, digest string) bool {
	r.Body = http.MaxBytesReader(nil, r.Body, 1048576) // Set max body size to 1MB

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return false
	}
	r.Body = ioutil.NopCloser(bytes.NewReader(body)) // Reset the request body

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write(body)
	expectedDigest := hex.EncodeToString(h.Sum(nil))

	return expectedDigest == digest
}
