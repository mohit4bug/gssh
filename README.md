# 🔑 gssh - Tiny GitHub account switcher

**Stop manually editing your ~/.ssh/config.**

![Built with Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge\&logo=go)
![TUI by Charm](https://img.shields.io/badge/TUI-Bubble%20Tea-F5A9B8?style=for-the-badge)

---

## What Is This?

A minimal TUI to manage multiple GitHub accounts.

It generates SSH keys and updates your SSH config so you can use multiple identities.

---

## Getting Started

```bash
curl -fsSL https://raw.githubusercontent.com/mohit4bug/gssh/main/install.sh | bash
```

Alternative:

```bash
go install github.com/mohit4bug/gssh@latest
```

Run:

```bash
gssh
```

---

## Keybindings

| Key   | Action          |
| ----- | --------------- |
| ↑/↓   | Navigate        |
| enter | Select account  |
| a     | Add new profile |
| e     | Edit alias      |
| d     | Delete profile  |
| k     | Show public key |
| q     | Quit            |

---

## Note

Modifies `~/.ssh/config`. Back it up if needed.
