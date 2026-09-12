# 🌐 omnichannel-hub

[![Release](https://img.shields.io/badge/Release-v1.0-6366f1.svg)](https://github.com/benzjeremy/omnichannel-hub/releases)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](https://github.com/benzjeremy/omnichannel-hub/blob/main/LICENSE)
[![Go: 1.22](https://img.shields.io/badge/Go-1.22-00ADD8.svg)](https://golang.org)
[![Security: Zero-Dummy](https://img.shields.io/badge/Security-Zero--Dummy--Standard-10b981.svg)](https://benzjeremy.github.io/omnichannel-hub/)
[![Isolation: Localhost](https://img.shields.io/badge/Isolation-127.0.0.1%20Only-38bdf8.svg)](https://benzjeremy.github.io/)

> **Unified Messaging Hub & Self-Hosted Communication Broker in Go 1.22**  
> Centralized, self-hosted communication router bundling Email (IMAP/SMTP TLS), WhatsApp, and Discord with zero cloud relays and hardened AES-256-GCM encryption.

---

## ⚡ Why omnichannel-hub?

Modern communication channels like Email, WhatsApp, and Discord are heavily fragmented. Traditional multi-messenger solutions rely on resource-hungry Electron instances containing dozens of embedded iframes, or route private messages through third-party cloud infrastructure.

`omnichannel-hub` offers a radically lightweight, secure alternative:
1. **Zero Cloud Relays:** No storage on third-party cloud servers. All messages and credentials reside encrypted on your own machine.
2. **Unified Message Broker:** Decoupled architecture utilizing Go channels for real-time aggregation and dispatching.
3. **Cross-Platform:** Unified abstraction layer for Email (IMAP/SMTP), Discord Bot Gateway, and WhatsApp Web Bridge.
4. **Resource-Efficient:** Minimal memory footprint (< 30 MB RAM) written in pure Go without forced CGO dependencies.
5. **Zero-Dummy-Security:** AES-256-GCM vault with PBKDF2 (100,000 rounds), 32-byte tokens, anti-DNS-rebinding, and anti-CSRF protection.

---

## 🛡️ Zero-Dummy-Security (Production-Hardened by Default)

- **AES-256-GCM Encryption:** All stored messages, accounts, and credentials/tokens are cryptographically protected in `hub_vault.enc`.
- **PBKDF2 Key Derivation:** At least **100,000 iterations** with SHA-256 and a 32-byte cryptographic random salt.
- **Strict Localhost Isolation:** Binds strictly to `127.0.0.1:8082`. No exposure to external networks without an explicit reverse proxy.
- **Cryptographic Token Authentication:** Every API request requires a 32-byte CSPRNG token (`X-Hub-Token`).
- **Anti-DNS-Rebinding & Anti-CSRF:** Strict validation of `Host` and `Origin` headers.

---

## 📦 Multi-Platform Installation & Download

### Via Go Install:
```bash
go install github.com/benzjeremy/omnichannel-hub@latest
```

### Linux (x86_64):
```bash
tar -xzf omnichannel-hub-v1.0-linux.tar.gz
sudo cp omnichannel-hub /usr/local/bin/
omnichannel-hub --email-address benzjeremy@pm.me
```

### Windows (x86_64):
Extract `omnichannel-hub-v1.0-windows.zip` and run `omnichannel-hub.exe`.

---

## 🚀 CLI Flags

```text
Usage of omnichannel-hub:
  --port int             Port to listen on (default 8082, binds to 127.0.0.1)
  --token string         32-byte security token (auto-generated if omitted)
  --data-dir string      Directory for encrypted storage vault (default "data")
  --email-address string Default monitored email address (default "benzjeremy@pm.me")
  --discord-bot string   Default discord bot identifier (default "HubBot#0001")
  --whatsapp-phone str   Default WhatsApp phone number (default "+491701234567")
  --version              Print version and exit
```

---

## 🌐 REST API Endpoints

All endpoints (except `/health`) require the header `X-Hub-Token: <token>`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Service status, version info, and active channel states |
| `GET` | `/messages` | Retrieve messages (Filters: `channel`, `direction`, `unread`, `limit`) |
| `POST` | `/messages/send` | Send message via selected channel (`email`, `whatsapp`, `discord`) |
| `POST` | `/messages/read` | Mark message as read by ID |
| `GET` | `/channels` | Overview of registered channels and account configurations |
| `POST` | `/channels` | Register or update communication account credentials |
| `GET` | `/stats` | Live counters for inbound and outbound messages per channel |

---

## 👥 Contributors & Credits

- **Jeremy Benz** ([@benzjeremy](https://github.com/benzjeremy) & [@jbenz1706](https://github.com/jbenz1706)) – Project Founder & Lead Developer
- **AI Assistants (Pair Programming):** Google Antigravity & Claude Code
- © 2026 Jeremy Benz

---

## 📄 License

This project is licensed under the **GNU General Public License, Version 3 (GPL-3.0)**.  
See [LICENSE](LICENSE) for details.
