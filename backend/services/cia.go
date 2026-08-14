package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const ciaAPIBase = "http://127.0.0.1:8000"

type CIAService struct {
	client *http.Client
}

func NewCIAService() *CIAService {
	return &CIAService{client: &http.Client{}}
}

// Encrypt llama a POST /confidentiality/encrypt
func (s *CIAService) Encrypt(message string) (string, error) {
	body, _ := json.Marshal(map[string]string{"message": message})
	resp, err := s.client.Post(ciaAPIBase+"/confidentiality/encrypt", "application/json", bytes.NewReader(body))
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
	resp, err := s.client.Post(ciaAPIBase+"/confidentiality/decrypt", "application/json", bytes.NewReader(body))
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
	resp, err := s.client.Post(ciaAPIBase+"/integrity/sign", "application/json", bytes.NewReader(body))
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
	resp, err := s.client.Post(ciaAPIBase+"/integrity/verify", "application/json", bytes.NewReader(body))
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

// HealthCheck verifica que la CIA API esté disponible
func (s *CIAService) HealthCheck() error {
	resp, err := s.client.Get(ciaAPIBase + "/")
	if err != nil {
		return fmt.Errorf("CIA API unreachable: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}
