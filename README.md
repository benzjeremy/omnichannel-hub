# 🌐 omnichannel-hub

[![Release](https://img.shields.io/badge/Release-v1.0-6366f1.svg)](https://github.com/benzjeremy/omnichannel-hub/releases)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](https://github.com/benzjeremy/omnichannel-hub/blob/main/LICENSE)
[![Go: 1.22](https://img.shields.io/badge/Go-1.22-00ADD8.svg)](https://golang.org)
[![Security: Zero-Dummy](https://img.shields.io/badge/Security-Zero--Dummy--Standard-10b981.svg)](https://benzjeremy.github.io/omnichannel-hub/)
[![Isolation: Localhost](https://img.shields.io/badge/Isolation-127.0.0.1%20Only-38bdf8.svg)](https://benzjeremy.github.io/)

> **Unified Messaging Hub & Self-Hosted Communication Broker in Go 1.22**  
> Zentraler, selbstgehosteter Nachrichten-Knotenpunkt zur Bündelung von E-Mail (IMAP/SMTP TLS), WhatsApp und Discord mit Zero Cloud Relays und militärisch gehärteter AES-256-GCM Verschlüsselung.

---

## ⚡ Warum omnichannel-hub?

Kommunikationskanäle wie E-Mail, WhatsApp und Discord sind fragmentiert. Herkömmliche Multi-Messenger-Lösungen basieren oft auf ressourcenhungrigen Electron-Instanzen mit Dutzenden eingebetteten Iframes oder leiten sensible Nachrichten über Drittanbieter-Cloud-Server um.

`omnichannel-hub` bietet eine radikal schlanke, sichere Alternative:
1. **Zero Cloud Relays:** Keine Speicherung auf Drittanbieter-Servern. Alle Nachrichten und Credentials liegen verschlüsselt auf dem eigenen System.
2. **Unified Message Broker:** Entkoppelte Architektur mit Go-Channels zur Aggregation und Weiterleitung von Nachrichten in Echtzeit.
3. **Plattformübergreifend:** Einheitliche Abstraktionsschicht für E-Mail (IMAP/SMTP), Discord Bot Gateway und WhatsApp Web Bridge.
4. **Ressourceneffizient:** Minimaler Speicher-Footprint (< 30 MB RAM) in purem Go ohne CGO-Zwang.
5. **Echte Sicherheit (Zero-Dummy-Security):** AES-256-GCM Vault mit PBKDF2 (100.000 Runden), 32-Byte Tokens, Anti-DNS-Rebinding und Anti-CSRF Schutz.

---

## 🛡️ Echte Sicherheit aus dem Effeff (Zero-Dummy-Security)

- **AES-256-GCM Verschlüsselung:** Alle gespeicherten Nachrichten, Konten und sensitive Passwörter/Tokens werden in `hub_vault.enc` kryptografisch geschützt.
- **PBKDF2 Key Derivation:** Mindestens **100.000 Iterationen** mit SHA-256 und 32-Byte kryptografischem Zufallssalt.
- **Strict Localhost Isolation:** Bindung ausschließlich an `127.0.0.1:8082`. Keine Exposition im Netzwerk ohne expliziten Reverse Proxy.
- **Kryptografische Authentifizierung:** Jeder API-Aufruf erfordert ein 32-Byte CSPRNG Token (`X-Hub-Token`).
- **Anti-DNS-Rebinding & Anti-CSRF:** Strikte Validierung des `Host`- und `Origin`-Headers.

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
Entpacke `omnichannel-hub-v1.0-windows.zip` und starte `omnichannel-hub.exe`.

---

## 🚀 CLI-Flags

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

## 🌐 REST API Endpunkte

Alle Endpunkte (außer `/health`) erfordern den Header `X-Hub-Token: <token>`.

| Methode | Pfad | Beschreibung |
|---|---|---|
| `GET` | `/health` | Service-Status, Versionsinfo und aktive Kanäle |
| `GET` | `/messages` | Abruf aller Nachrichten (Filter: `channel`, `direction`, `unread`, `limit`) |
| `POST` | `/messages/send` | Nachricht über einen beliebigen Kanal versenden (`email`, `whatsapp`, `discord`) |
| `POST` | `/messages/read` | Nachricht anhand ihrer ID als gelesen markieren |
| `GET` | `/channels` | Übersicht aller registrierten Kanäle und Account-Konfigurationen |
| `POST` | `/channels` | Neues Kommunikationskonto anlegen oder aktualisieren |
| `GET` | `/stats` | Live-Zähler für ein- und ausgehende Nachrichten pro Kanal |

---

## 👥 Mitwirkende & Credits

- **Jeremy Benz** ([@benzjeremy](https://github.com/benzjeremy) & [@jbenz1706](https://github.com/jbenz1706)) – Projektgründer & Lead Developer
- **AI-Assistenten (Pair Programming):** Google Antigravity & Claude Code
- © 2026 Jeremy Benz

---

## 📄 Lizenz

Dieses Projekt steht unter der **GNU General Public License, Version 3 (GPL-3.0)**.  
Weitere Informationen: [Offizielle Lizenz (GPL-3.0)](https://github.com/benzjeremy/omnichannel-hub/blob/main/LICENSE)
