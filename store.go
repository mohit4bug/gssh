package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Account struct {
	Alias    string `json:"alias"`
	Username string `json:"username"`
	Email    string `json:"email"`
	KeyPath  string `json:"key_path"`
	Active   bool   `json:"active"`
}

func homeDir() string   { h, _ := os.UserHomeDir(); return h }
func gsshDir() string   { return filepath.Join(homeDir(), ".gssh") }
func sshDir() string    { return filepath.Join(homeDir(), ".ssh") }
func storeFile() string { return filepath.Join(gsshDir(), "accounts.json") }

func loadAccounts() ([]Account, error) {
	b, err := os.ReadFile(storeFile())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var a []Account
	return a, json.Unmarshal(b, &a)
}

func saveAccounts(a []Account) error {
	if err := os.MkdirAll(gsshDir(), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(storeFile(), b, 0600)
}

func createAccount(existing []Account, alias, username, email string) ([]Account, string, error) {
	for _, a := range existing {
		if a.Alias == alias {
			return nil, "", fmt.Errorf("alias %q already exists", alias)
		}
	}

	if err := os.MkdirAll(sshDir(), 0700); err != nil {
		return nil, "", err
	}

	keyPath := filepath.Join(sshDir(), "gssh_"+alias)
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		out, err := exec.Command(
			"ssh-keygen", "-t", "ed25519",
			"-f", keyPath, "-N", "", "-C", email,
		).CombinedOutput()
		if err != nil {
			return nil, "", fmt.Errorf("ssh-keygen: %s", strings.TrimSpace(string(out)))
		}
	}

	if err := sshConfigAdd(alias, keyPath); err != nil {
		return nil, "", err
	}

	accounts := append(existing, Account{
		Alias:    alias,
		Username: username,
		Email:    email,
		KeyPath:  keyPath,
	})

	pubKey, err := readPubKey(keyPath)
	if err != nil {
		return nil, "", err
	}

	return accounts, pubKey, saveAccounts(accounts)
}

func activate(accounts []Account, idx int) ([]Account, error) {
	for i := range accounts {
		accounts[i].Active = i == idx
	}
	a := accounts[idx]
	if err := gitSet("user.name", a.Username); err != nil {
		return nil, err
	}
	if err := gitSet("user.email", a.Email); err != nil {
		return nil, err
	}
	return accounts, saveAccounts(accounts)
}

func deleteAcc(accounts []Account, idx int) ([]Account, error) {
	a := accounts[idx]
	sshConfigRemove(a.Alias)
	os.Remove(a.KeyPath)
	os.Remove(a.KeyPath + ".pub")
	accounts = append(accounts[:idx], accounts[idx+1:]...)
	return accounts, saveAccounts(accounts)
}

func editAlias(accounts []Account, idx int, newAlias string) ([]Account, error) {
	oldAlias := accounts[idx].Alias
	if oldAlias == newAlias {
		return accounts, nil
	}

	for i, a := range accounts {
		if i != idx && a.Alias == newAlias {
			return nil, fmt.Errorf("alias %q already exists", newAlias)
		}
	}

	oldKeyPath := accounts[idx].KeyPath
	newKeyPath := filepath.Join(sshDir(), "gssh_"+newAlias)

	if _, err := os.Stat(oldKeyPath); err == nil {
		os.Rename(oldKeyPath, newKeyPath)
		os.Rename(oldKeyPath+".pub", newKeyPath+".pub")
	}

	sshConfigRemove(oldAlias)
	if err := sshConfigAdd(newAlias, newKeyPath); err != nil {
		return nil, err
	}

	accounts[idx].Alias = newAlias
	accounts[idx].KeyPath = newKeyPath

	return accounts, saveAccounts(accounts)
}

func readPubKey(keyPath string) (string, error) {
	b, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return "", fmt.Errorf("cannot read public key: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

func gitSet(key, val string) error {
	out, err := exec.Command("git", "config", "--global", key, val).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git config %s: %s", key, strings.TrimSpace(string(out)))
	}
	return nil
}

const mark = "# gssh:"

func sshConfigPath() string { return filepath.Join(sshDir(), "config") }

func sshConfigRead() string {
	b, _ := os.ReadFile(sshConfigPath())
	return string(b)
}

func sshConfigAdd(alias, keyPath string) error {
	block := fmt.Sprintf(
		"%s%s\nHost github-%s\n  HostName github.com\n  User git\n  IdentityFile %s\n  AddKeysToAgent yes\n  IdentitiesOnly yes\n%s%s:end\n",
		mark, alias, alias, keyPath, mark, alias,
	)
	content := sshConfigStrip(sshConfigRead(), alias)
	content = strings.TrimRight(content, "\n") + "\n\n" + block
	return os.WriteFile(sshConfigPath(), []byte(content), 0600)
}

func sshConfigRemove(alias string) {
	os.WriteFile(sshConfigPath(), []byte(sshConfigStrip(sshConfigRead(), alias)), 0600)
}

func sshConfigStrip(content, alias string) string {
	start := mark + alias + "\n"
	end := mark + alias + ":end\n"
	before, _, ok := strings.Cut(content, start)
	if !ok {
		return content
	}
	_, after, ok := strings.Cut(content, end)
	if !ok {
		return content
	}
	return before + after
}
