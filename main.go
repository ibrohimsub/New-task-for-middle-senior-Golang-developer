package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
)

type Wallet struct {
	ID       string
	Balance  float64
	Identified  bool
}

var wallets = []Wallet{
	{ID: "1", Balance: 10000, Identified: false},
	{ID: "2", Balance: 100000, Identified: true},
	// Добавьте другие записи кошельков по вашему усмотрению
}

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
		// Проверка наличия кошелька по userId
		// Возвращение результата в формате JSON
	} else {
		http.Error(w, "Invalid digest", http.StatusUnauthorized)
	}
}

func depositToWalletHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-UserId")
	digest := r.Header.Get("X-Digest")

	if validateDigest(r.Body, digest) {
		// Пополнение кошелька
		// Возвращение результата в формате JSON
	} else {
		http.Error(w, "Invalid digest", http.StatusUnauthorized)
	}
}

func getMonthlyOperationsHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-UserId")
	digest := r.Header.Get("X-Digest")

	if validateDigest(r.Body, digest) {
		// Получение общего количества и суммы операций пополнения за текущий месяц
		// Возвращение результата в формате JSON
	} else {
		http.Error(w, "Invalid digest", http.StatusUnauthorized)
	}
}

func getWalletBalanceHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-UserId")
	digest := r.Header.Get("X-Digest")

	if validateDigest(r.Body, digest) {
		// Получение баланса кошелька
		// Возвращение результата в формате JSON
	} else {
		http.Error(w, "Invalid digest", http.StatusUnauthorized)
	}
}

func validateDigest(data []byte, digest string) bool {
	h := hmac.New(sha1.New, []byte("your-secret-key"))
	h.Write(data)
	expectedDigest := hex.EncodeToString(h.Sum(nil))

	return expectedDigest == digest
}
