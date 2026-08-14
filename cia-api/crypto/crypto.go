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
	FernetKey string `json:"fernet_key"`
	RSAKey    string `json:"rsa_key"`
}

type CryptoService struct {
	mu         sync.RWMutex
	fernetKey  *fernet.Key
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
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

	// Load Fernet key
	decoded, err := base64.StdEncoding.DecodeString(saved.FernetKey)
	if err != nil || len(decoded) != 32 {
		return nil, fmt.Errorf("invalid fernet key")
	}
	var k fernet.Key
	copy(k[:], decoded)

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
		fernetKey:  &k,
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}, nil
}

func (s *CryptoService) saveKeys() error {
	// Encode Fernet key
	fernetB64 := base64.StdEncoding.EncodeToString(s.fernetKey[:])

	// Encode RSA private key as PEM
	privBytes := x509.MarshalPKCS1PrivateKey(s.privateKey)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})

	saved := savedKeys{
		FernetKey: fernetB64,
		RSAKey:    string(privPEM),
	}

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(keysFile, data, 0600)
}

// Encrypt cifra un mensaje usando Fernet (AES-CBC + HMAC)
func (s *CryptoService) Encrypt(plaintext string) (string, error) {
	tok, err := fernet.EncryptAndSign([]byte(plaintext), s.fernetKey)
	if err != nil {
		return "", fmt.Errorf("fernet encrypt failed: %w", err)
	}
	return string(tok), nil
}

// Decrypt descifra un mensaje Fernet
func (s *CryptoService) Decrypt(ciphertext string) (string, error) {
	msg := fernet.VerifyAndDecrypt([]byte(ciphertext), 0, []*fernet.Key{s.fernetKey})
	if msg == nil {
		return "", fmt.Errorf("invalid or expired ciphertext")
	}
	return string(msg), nil
}

// Sign firma un mensaje usando RSA-SHA256
func (s *CryptoService) Sign(message string) (string, error) {
	hash := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// Verify verifica la firma de un mensaje
func (s *CryptoService) Verify(message, signature string) (bool, error) {
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	hash := sha256.Sum256([]byte(message))
	err = rsa.VerifyPKCS1v15(s.publicKey, crypto.SHA256, hash[:], sigBytes)
	return err == nil, nil
}

// ExportPublicKey exporta la clave pública en formato PEM
func (s *CryptoService) ExportPublicKey() string {
	pubASN1, _ := x509.MarshalPKIXPublicKey(s.publicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubASN1,
	})
	return string(pubPEM)
}
