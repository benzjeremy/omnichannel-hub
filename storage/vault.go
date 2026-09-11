package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/benzjeremy/omnichannel-hub/models"
	"golang.org/x/crypto/pbkdf2"
)

const (
	PBKDF2Iterations = 100000
	KeySize          = 32
	SaltSize         = 32
	NonceSize        = 12
)

type EncryptedPayload struct {
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

type VaultData struct {
	Messages []models.Message `json:"messages"`
	Accounts []models.Account `json:"accounts"`
}

type Vault struct {
	mu       sync.RWMutex
	filePath string
	key      []byte
	data     VaultData
}

func DeriveKey(passphrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, PBKDF2Iterations, KeySize, sha256.New)
}

func Encrypt(key, plaintext []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gcm: %w", err)
	}
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return nonce, ciphertext, nil
}

func Decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	return plaintext, nil
}

func OpenVault(filePath, passphrase string) (*Vault, error) {
	if passphrase == "" {
		return nil, errors.New("passphrase cannot be empty")
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed creating vault directory: %w", err)
	}

	v := &Vault{
		filePath: filePath,
		data: VaultData{
			Messages: make([]models.Message, 0),
			Accounts: make([]models.Account, 0),
		},
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		salt := make([]byte, SaltSize)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return nil, fmt.Errorf("failed to generate salt: %w", err)
		}
		v.key = DeriveKey(passphrase, salt)
		if err := v.saveWithSalt(salt); err != nil {
			return nil, err
		}
		return v, nil
	}

	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read vault file: %w", err)
	}

	var payload EncryptedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("corrupted vault structure: %w", err)
	}

	v.key = DeriveKey(passphrase, payload.Salt)
	plaintext, err := Decrypt(v.key, payload.Nonce, payload.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt vault: %w", err)
	}

	if err := json.Unmarshal(plaintext, &v.data); err != nil {
		return nil, fmt.Errorf("failed parsing vault data: %w", err)
	}

	return v, nil
}

func (v *Vault) saveWithSalt(salt []byte) error {
	marshaled, err := json.Marshal(v.data)
	if err != nil {
		return fmt.Errorf("failed to marshal vault data: %w", err)
	}

	nonce, ciphertext, err := Encrypt(v.key, marshaled)
	if err != nil {
		return err
	}

	payload := EncryptedPayload{
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(v.filePath, out, 0600)
}

func (v *Vault) saveLocked() error {
	raw, err := os.ReadFile(v.filePath)
	if err != nil {
		return fmt.Errorf("cannot read existing vault salt: %w", err)
	}

	var existing EncryptedPayload
	if err := json.Unmarshal(raw, &existing); err != nil {
		return fmt.Errorf("cannot parse existing vault salt: %w", err)
	}

	return v.saveWithSalt(existing.Salt)
}

// SaveMessage stores a message in the vault.
func (v *Vault) SaveMessage(msg models.Message) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if msg.ID == "" {
		msg.ID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	}

	v.data.Messages = append(v.data.Messages, msg)
	return v.saveLocked()
}

// GetMessages retrieves messages matching filter criteria.
func (v *Vault) GetMessages(filter models.MessageFilter) []models.Message {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var result []models.Message

	// Iterate backwards for newest first
	for i := len(v.data.Messages) - 1; i >= 0; i-- {
		m := v.data.Messages[i]

		if filter.Channel != "" && m.Channel != filter.Channel {
			continue
		}
		if filter.Direction != "" && m.Direction != filter.Direction {
			continue
		}
		if filter.UnreadOnly && m.Read {
			continue
		}

		result = append(result, m)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}

	return result
}

// MarkRead marks a message by ID as read.
func (v *Vault) MarkRead(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	found := false
	for i := range v.data.Messages {
		if v.data.Messages[i].ID == id {
			v.data.Messages[i].Read = true
			found = true
			break
		}
	}

	if !found {
		return errors.New("message not found")
	}

	return v.saveLocked()
}

// SaveAccount adds or updates an account configuration.
func (v *Vault) SaveAccount(acc models.Account) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	acc.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	updated := false
	for i := range v.data.Accounts {
		if v.data.Accounts[i].ID == acc.ID {
			v.data.Accounts[i] = acc
			updated = true
			break
		}
	}

	if !updated {
		v.data.Accounts = append(v.data.Accounts, acc)
	}

	return v.saveLocked()
}

// GetAccounts returns all configured accounts.
func (v *Vault) GetAccounts() []models.Account {
	v.mu.RLock()
	defer v.mu.RUnlock()

	out := make([]models.Account, len(v.data.Accounts))
	copy(out, v.data.Accounts)
	return out
}
