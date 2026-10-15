package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// loadSecret reads the key that encrypts users' API keys, creating it on first start. It sits in the data
// folder next to the database but outside the backup (only backup.db and files/ are copied off the server),
// so a backup alone does not reveal the API keys. Restoring on a new server without it means pasting the API keys again.
func loadSecret(dataDir string) (cipher.AEAD, error) {
	path := filepath.Join(dataDir, "secret.key")
	key, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		key = make([]byte, 32)
		rand.Read(key)
		err = os.WriteFile(path, key, 0o600)
	}
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("secret.key inválido")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (a *App) sealKey(plain string) string {
	if plain == "" {
		return ""
	}
	nonce := make([]byte, a.secret.NonceSize())
	rand.Read(nonce)
	return "v1:" + base64.StdEncoding.EncodeToString(a.secret.Seal(nonce, nonce, []byte(plain), nil))
}

func (a *App) openKey(stored string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, "v1:"))
	if stored == "" || err != nil || !strings.HasPrefix(stored, "v1:") || len(raw) < a.secret.NonceSize() {
		return "", err
	}
	n := a.secret.NonceSize()
	plain, err := a.secret.Open(nil, raw[:n], raw[n:], nil)
	return string(plain), err
}

// sealOldKeys encrypts API keys saved before encryption existed.
func (a *App) sealOldKeys() error {
	rows, err := a.db.Query(`SELECT id, anthropic_key FROM users WHERE anthropic_key != '' AND anthropic_key NOT LIKE 'v1:%'`)
	if err != nil {
		return err
	}
	old := map[int64]string{}
	for rows.Next() {
		var id int64
		var k string
		if err := rows.Scan(&id, &k); err != nil {
			rows.Close()
			return err
		}
		old[id] = k
	}
	rows.Close()
	for id, k := range old {
		if _, err := a.db.Exec(`UPDATE users SET anthropic_key = ? WHERE id = ?`, a.sealKey(k), id); err != nil {
			return err
		}
	}
	return nil
}
