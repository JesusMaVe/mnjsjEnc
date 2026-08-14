package main

import (
	"encoding/json"
	"log"
	"net/http"
	"securemessage-cia/availability"
	"securemessage-cia/crypto"
)

var (
	svc     = crypto.New()
	cluster = availability.NewCluster(3, 0.3)
)

type Request struct {
	Message    string `json:"message"`
	Ciphertext string `json:"ciphertext"`
	Signature  string `json:"signature"`
}

func respond(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func handleEncrypt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respond(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	ciphertext, err := svc.Encrypt(req.Message)
	if err != nil {
		log.Printf("encrypt error: %v", err)
		respond(w, http.StatusInternalServerError, map[string]string{"error": "encryption failed"})
		return
	}
	respond(w, http.StatusOK, map[string]string{"ciphertext": ciphertext})
}

func handleDecrypt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respond(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	plaintext, err := svc.Decrypt(req.Ciphertext)
	if err != nil {
		respond(w, http.StatusForbidden, map[string]string{"error": "invalid ciphertext"})
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": plaintext})
}

func handleSign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respond(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	signature, err := svc.Sign(req.Message)
	if err != nil {
		log.Printf("sign error: %v", err)
		respond(w, http.StatusInternalServerError, map[string]string{"error": "signing failed"})
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": req.Message, "signature": signature})
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respond(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	valid, err := svc.Verify(req.Message, req.Signature)
	if err != nil {
		log.Printf("verify error: %v", err)
		respond(w, http.StatusInternalServerError, map[string]string{"error": "verification failed"})
		return
	}
	respond(w, http.StatusOK, map[string]bool{"valid": valid})
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"message": "CIA Triad API running (Go)"})
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func main() {
	http.HandleFunc("/confidentiality/encrypt", cors(handleEncrypt))
	http.HandleFunc("/confidentiality/decrypt", cors(handleDecrypt))
	http.HandleFunc("/integrity/sign", cors(handleSign))
	http.HandleFunc("/integrity/verify", cors(handleVerify))
	http.HandleFunc("/availability/status", cors(cluster.HandleStatus))
	http.HandleFunc("/availability/request", cors(cluster.HandleRequest))
	http.HandleFunc("/", cors(handleRoot))

	log.Println("CIA API running on :8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}
