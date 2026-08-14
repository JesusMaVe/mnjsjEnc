package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func getCIAAPIBase() string {
	if v := os.Getenv("CIA_API_BASE_URL"); v != "" {
		return v
	}
	return "http://127.0.0.1:8000"
}

type CIAService struct {
	client *http.Client
}

func NewCIAService() *CIAService {
	// For self-signed certs in dev, skip verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // dev only
	}
	return &CIAService{client: &http.Client{Transport: tr}}
}

// Encrypt llama a POST /confidentiality/encrypt
func (s *CIAService) Encrypt(message string) (string, error) {
	body, _ := json.Marshal(map[string]string{"message": message})
	resp, err := s.client.Post(getCIAAPIBase()+"/confidentiality/encrypt", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("encrypt request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Ciphertext string `json:"ciphertext"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("encrypt decode failed: %w", err)
	}
	return result.Ciphertext, nil
}

// Decrypt llama a POST /confidentiality/decrypt
func (s *CIAService) Decrypt(ciphertext string) (string, error) {
	body, _ := json.Marshal(map[string]string{"ciphertext": ciphertext})
	resp, err := s.client.Post(getCIAAPIBase()+"/confidentiality/decrypt", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("decrypt request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("invalid ciphertext")
	}

	var result struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decrypt decode failed: %w", err)
	}
	return result.Message, nil
}

// Sign llama a POST /integrity/sign
func (s *CIAService) Sign(message string) (string, error) {
	body, _ := json.Marshal(map[string]string{"message": message})
	resp, err := s.client.Post(getCIAAPIBase()+"/integrity/sign", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("sign request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Signature string `json:"signature"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("sign decode failed: %w", err)
	}
	return result.Signature, nil
}

// Verify llama a POST /integrity/verify
func (s *CIAService) Verify(message, signature string) (bool, error) {
	body, _ := json.Marshal(map[string]string{"message": message, "signature": signature})
	resp, err := s.client.Post(getCIAAPIBase()+"/integrity/verify", "application/json", bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("verify request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("verify decode failed: %w", err)
	}
	return result.Valid, nil
}

// HealthCheck verifica que la CIA API este disponible
func (s *CIAService) HealthCheck() error {
	resp, err := s.client.Get(getCIAAPIBase() + "/")
	if err != nil {
		return fmt.Errorf("CIA API unreachable: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}
