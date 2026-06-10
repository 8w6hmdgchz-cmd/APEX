package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// AES256GCM encrypts/decrypts using AES-256-GCM.
type AES256GCM struct {
	key []byte
}

// NewAES256GCM creates a new AES-256-GCM cipher. Key must be 32 bytes.
func NewAES256GCM(key []byte) (*AES256GCM, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
	}
	return &AES256GCM{key: key}, nil
}

// Encrypt encrypts plaintext.
func (a *AES256GCM) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext.
func (a *AES256GCM) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(ciphertext) < ns {
		return nil, errors.New("ciphertext too short")
	}
	return gcm.Open(nil, ciphertext[:ns], ciphertext[ns:], nil)
}

// EncryptString encrypts and base64-encodes a string.
func (a *AES256GCM) EncryptString(plaintext string) (string, error) {
	enc, err := a.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(enc), nil
}

// DecryptString base64-decodes and decrypts a string.
func (a *AES256GCM) DecryptString(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	dec, err := a.Decrypt(data)
	if err != nil {
		return "", err
	}
	return string(dec), nil
}

// ECDSAKeyPair holds an ECDSA key pair.
type ECDSAKeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

// GenerateECDSAKey generates a new ECDSA P-256 key pair.
func GenerateECDSAKey() (*ECDSAKeyPair, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &ECDSAKeyPair{PrivateKey: key, PublicKey: &key.PublicKey}, nil
}

// Sign signs data with the private key.
func (kp *ECDSAKeyPair) Sign(data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, kp.PrivateKey, hash[:])
	if err != nil {
		return nil, err
	}
	sig := append(r.Bytes(), s.Bytes()...)
	return sig, nil
}

// Verify verifies a signature with the public key.
func Verify(pub *ecdsa.PublicKey, data, sig []byte) bool {
	hash := sha256.Sum256(data)
	rLen := len(sig) / 2
	r := new(big.Int).SetBytes(sig[:rLen])
	s := new(big.Int).SetBytes(sig[rLen:])
	return ecdsa.Verify(pub, hash[:], r, s)
}

// ExportPublicKey exports a public key to PEM-like base64.
func ExportPublicKey(pub *ecdsa.PublicKey) (string, error) {
	b, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// JWTClaims represents JWT claims.
type JWTClaims struct {
	Subject   string `json:"sub"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	Role      string `json:"role"`
}

// JWT manages JWT token creation and validation.
type JWT struct {
	secret []byte
	expiry time.Duration
}

// NewJWT creates a new JWT manager.
func NewJWT(secret string, expiry time.Duration) *JWT {
	return &JWT{secret: []byte(secret), expiry: expiry}
}

// Generate creates a new JWT token.
func (j *JWT) Generate(subject, role string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		Subject:   subject,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(j.expiry).Unix(),
		Role:      role,
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	p := base64.RawURLEncoding.EncodeToString(payload)
	data := header + "." + p
	sig := j.sign(data)
	return data + "." + sig, nil
}

// Validate validates a JWT token and returns claims.
func (j *JWT) Validate(token string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}
	data := parts[0] + "." + parts[1]
	expectedSig := j.sign(data)
	if expectedSig != parts[2] {
		return nil, errors.New("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims JWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}

func (j *JWT) sign(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	h.Write(j.secret)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// HashPassword creates a SHA-256 hash of a password (simplified).
func HashPassword(password string) string {
	h := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", h)
}
