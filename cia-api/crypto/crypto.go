// Package crypto provides Fernet encryption and RSA-PSS signing.
package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/fernet/fernet-go"
)

const keysDir = "keys"
const keysFile = "keys/crypto.json"

type savedKeys struct {
	FernetKeys []string `json:"fernet_keys"` // keyring: current + previous
	RSAKey     string   `json:"rsa_key"`
}

type CryptoService struct {
	mu          sync.RWMutex
	fernetKey   *fernet.Key   // current encryption key
	fernetKeys  []*fernet.Key // keyring for decryption (current + previous)
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
}

func New() *CryptoService {
	_ = os.MkdirAll(keysDir, 0700)

	if keys, err := loadKeys(); err == nil {
		log.Println("Loaded existing keys from disk")
		return keys
	}

	log.Println("Generating new keys (first run)")
	return generateAndSaveKeys()
}

func generateAndSaveKeys() *CryptoService {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	var k fernet.Key
	k.Generate()

	svc := &CryptoService{
		fernetKey:  &k,
		fernetKeys: []*fernet.Key{&k},
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}

	if err := svc.saveKeys(); err != nil {
		log.Printf("WARNING: could not save keys: %v", err)
	}

	return svc
}

func loadKeys() (*CryptoService, error) {
	data, err := os.ReadFile(keysFile)
	if err != nil {
		return nil, err
	}

	var saved savedKeys
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("corrupted keys file")
	}

	// Load Fernet keyring
	if len(saved.FernetKeys) == 0 {
		return nil, fmt.Errorf("no fernet keys saved")
	}
	var fernetKeys []*fernet.Key
	for _, k64 := range saved.FernetKeys {
		decoded, err := base64.StdEncoding.DecodeString(k64)
		if err != nil || len(decoded) != 32 {
			return nil, fmt.Errorf("invalid fernet key in keyring")
		}
		var k fernet.Key
		copy(k[:], decoded)
		fernetKeys = append(fernetKeys, &k)
	}

	// Load RSA private key
	block, _ := pem.Decode([]byte(saved.RSAKey))
	if block == nil {
		return nil, fmt.Errorf("invalid RSA key PEM")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid RSA key: %w", err)
	}

	return &CryptoService{
		fernetKey:  fernetKeys[0], // first key is current
		fernetKeys: fernetKeys,
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}, nil
}

func (s *CryptoService) saveKeys() error {
	// Encode Fernet keyring
	var fernetB64s []string
	for _, k := range s.fernetKeys {
		fernetB64s = append(fernetB64s, base64.StdEncoding.EncodeToString(k[:]))
	}

	// Encode RSA private key as PEM
	privBytes := x509.MarshalPKCS1PrivateKey(s.privateKey)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})

	saved := savedKeys{
		FernetKeys: fernetB64s,
		RSAKey:     string(privPEM),
	}

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(keysFile, data, 0600)
}

// Encrypt cifra un mensaje usando Fernet (AES-CBC + HMAC)
func (s *CryptoService) Encrypt(plaintext string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tok, err := fernet.EncryptAndSign([]byte(plaintext), s.fernetKey)
	if err != nil {
		return "", fmt.Errorf("encryption failed")
	}
	return string(tok), nil
}

// Decrypt descifra un mensaje Fernet — tries all keys in keyring
func (s *CryptoService) Decrypt(ciphertext string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msg := fernet.VerifyAndDecrypt([]byte(ciphertext), 0, s.fernetKeys)
	if msg == nil {
		return "", fmt.Errorf("invalid or expired ciphertext")
	}
	return string(msg), nil
}

// Sign firma un mensaje usando RSA-PSS (more secure than PKCS1v15)
func (s *CryptoService) Sign(message string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hash := sha256.Sum256([]byte(message))
	opts := &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthAuto}
	signature, err := rsa.SignPSS(rand.Reader, s.privateKey, crypto.SHA256, hash[:], opts)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// Verify verifica la firma de un mensaje usando RSA-PSS
func (s *CryptoService) Verify(message, signature string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	hash := sha256.Sum256([]byte(message))
	opts := &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthAuto}
	err = rsa.VerifyPSS(s.publicKey, crypto.SHA256, hash[:], sigBytes, opts)
	return err == nil, nil
}

// RotateKey genera una nueva llave Fernet y archiva la anterior
func (s *CryptoService) RotateKey() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Archive current key
	s.fernetKeys = append([]*fernet.Key{s.fernetKey}, s.fernetKeys...)
	// Keep only 2 keys (current + 1 previous)
	if len(s.fernetKeys) > 2 {
		s.fernetKeys = s.fernetKeys[:2]
	}

	// Generate new key
	var k fernet.Key
	k.Generate()
	s.fernetKey = &k
	s.fernetKeys[0] = &k

	log.Println("Fernet key rotated successfully")
	return s.saveKeys()
}

// ExportPublicKey exporta la clave publica en formato PEM
func (s *CryptoService) ExportPublicKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pubASN1, _ := x509.MarshalPKIXPublicKey(s.publicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubASN1,
	})
	return string(pubPEM)
}
