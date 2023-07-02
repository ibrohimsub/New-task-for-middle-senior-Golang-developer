package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Wallet struct {
	ID         string
	Balance    float64
	Identified bool
}

var wallets = map[string]Wallet{
	"1": {ID: "1", Balance: 10000, Identified: false},
	"2": {ID: "2", Balance: 100000, Identified: true},
}

const (
	maxBalanceIdentified   = 100000.0
	maxBalanceUnidentified = 10000.0
	secretKey              = "your-secret-key"
)

func main() {
	http.HandleFunc("/wallet/check", checkWalletExistsHandler)
	http.HandleFunc("/wallet/deposit", depositToWalletHandler)
	http.HandleFunc("/wallet/operations", getMonthlyOperationsHandler)
	http.HandleFunc("/wallet/balance", getWalletBalanceHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func checkWalletExistsHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-UserId")
	digest := r.Header.Get("X-Digest")

	if validateDigest(r.Body, digest) {
		wallet, found := wallets[userId]
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

	if validateDigest(r.Body, digest) {
		var depositAmount float64
		err := json.NewDecoder(r.Body).Decode(&depositAmount)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		wallet, found := wallets[userId]
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
		wallets[userId] = wallet

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

	if validateDigest(r.Body, digest) {
		// Здесь можно реализовать логику для получения общего количества и суммы операций пополнения за текущий месяц
		// Возвращение результата в формате JSON
	} else {
		http.Error(w, "Invalid digest", http.StatusUnauthorized)
	}
}

func getWalletBalanceHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-UserId")
	digest := r.Header.Get("X-Digest")

	if validateDigest(r.Body, digest) {
		wallet, found := wallets[userId]
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

func validateDigest(data []byte, digest string) bool {
	h := hmac.New(sha1.New, []byte(secretKey))
	h.Write(data)
	expectedDigest := hex.EncodeToString(h.Sum(nil))

	return expectedDigest == digest
}