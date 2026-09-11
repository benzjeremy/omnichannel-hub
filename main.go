package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/benzjeremy/omnichannel-hub/api"
	"github.com/benzjeremy/omnichannel-hub/broker"
	"github.com/benzjeremy/omnichannel-hub/collectors"
	"github.com/benzjeremy/omnichannel-hub/storage"
)

const (
	Version = "v1.0"
	Banner  = `
  ██████╗ ███╗   ███╗███╗   ██╗██╗ ██████╗██╗  ██╗ █████╗ ███╗   ██╗███╗   ██╗███████╗██╗     
 ██╔═══██╗████╗ ████║████╗  ██║██║██╔════╝██║  ██║██╔══██╗████╗  ██║████╗  ██║██╔════╝██║     
 ██║   ██║██╔████╔██║██╔██╗ ██║██║██║     ███████║███████║██╔██╗ ██║██╔██╗ ██║█████╗  ██║     
 ██║   ██║██║╚██╔╝██║██║╚██╗██║██║██║     ██╔══██║██╔══██║██║╚██╗██║██║╚██╗██║██╔══╝  ██║     
 ╚██████╔╝██║ ╚═╝ ██║██║ ╚████║██║╚██████╗██║  ██║██║  ██║██║ ╚████║██║ ╚████║███████╗███████╗
  ╚═════╝ ╚═╝     ╚═╝╚═╝  ╚═══╝╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝  ╚═══╝╚══════╝╚══════╝
                     UNIFIED MESSAGING HUB & REAL-TIME BROKER (Go 1.22)
`
)

func main() {
	portFlag := flag.Int("port", 8082, "Port to listen on (binds exclusively to 127.0.0.1)")
	tokenFlag := flag.String("token", "", "32-byte security token (auto-generated if omitted)")
	dataDirFlag := flag.String("data-dir", "data", "Directory for encrypted storage vault")
	emailAddrFlag := flag.String("email-address", "benzjeremy@pm.me", "Default monitored email address")
	discordBotFlag := flag.String("discord-bot", "HubBot#0001", "Default discord bot identifier")
	waPhoneFlag := flag.String("whatsapp-phone", "+491701234567", "Default WhatsApp phone number")
	versionFlag := flag.Bool("version", false, "Print version and exit")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("omnichannel-hub %s (Jeremy Benz Zero-Dummy-Security)\n", Version)
		os.Exit(0)
	}

	fmt.Print(Banner)
	fmt.Println("================================================================================")
	fmt.Printf(" [SYSTEM] Version:       %s\n", Version)
	fmt.Printf(" [SYSTEM] Host Bind:     127.0.0.1:%d (Strict Localhost Isolation)\n", *portFlag)
	fmt.Printf(" [CHAN]   Email Bridge:  %s (IMAP/SMTP TLS)\n", *emailAddrFlag)
	fmt.Printf(" [CHAN]   Discord Bot:   %s (Bot Gateway)\n", *discordBotFlag)
	fmt.Printf(" [CHAN]   WhatsApp:      %s (Baileys Bridge)\n", *waPhoneFlag)
	fmt.Println("================================================================================")

	// 1. Initialize encrypted vault storage
	if err := os.MkdirAll(*dataDirFlag, 0700); err != nil {
		log.Fatalf("[-] Failed to create data directory: %v", err)
	}
	vaultPath := filepath.Join(*dataDirFlag, "hub_vault.enc")
	vaultPassphrase := os.Getenv("HUB_VAULT_KEY")
	if vaultPassphrase == "" {
		vaultPassphrase = "DefaultJeremyBenzOmnichannelHubEncryptedKey2026!"
	}

	vault, err := storage.OpenVault(vaultPath, vaultPassphrase)
	if err != nil {
		log.Fatalf("[-] Failed to initialize encrypted vault: %v", err)
	}
	fmt.Printf(" [SEC]    Encrypted Vault: %s (AES-256-GCM + PBKDF2 100k rounds)\n", vaultPath)

	// 2. Initialize Broker & Collectors Manager
	brk := broker.NewBroker(vault)
	mgr := collectors.NewManager(brk)

	emailCollector := collectors.NewEmailCollector("email-primary", *emailAddrFlag, brk)
	discordCollector := collectors.NewDiscordCollector("discord-primary", *discordBotFlag, brk)
	waCollector := collectors.NewWhatsAppCollector("wa-primary", *waPhoneFlag, brk)

	mgr.Register(emailCollector)
	mgr.Register(discordCollector)
	mgr.Register(waCollector)

	collectorCtx, cancelCollectors := context.WithCancel(context.Background())
	mgr.StartAll(collectorCtx)
	fmt.Println(" [✓] Message Collectors: ACTIVE (Email, WhatsApp, Discord)")

	// 3. Initialize Secure REST API Server
	cfg := api.ServerConfig{
		Port:    *portFlag,
		Token:   *tokenFlag,
		Version: Version,
	}

	server, err := api.NewServer(cfg, vault, brk, mgr)
	if err != nil {
		log.Fatalf("[-] Failed configuring API server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("[-] Failed starting API server: %v", err)
	}

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf(" [SEC] API Security Token: %s\n", server.Token())
	fmt.Println("       Pass as Header:   'X-Hub-Token: <token>'")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println(" [API] Available Endpoints:")
	fmt.Println("       GET  /health          - Service status and active channels")
	fmt.Println("       GET  /messages        - List all aggregated messages (filter by channel/unread)")
	fmt.Println("       POST /messages/send   - Send outbound message across any channel")
	fmt.Println("       POST /messages/read   - Mark message as read")
	fmt.Println("       GET  /channels        - List registered channels & accounts")
	fmt.Println("       GET  /stats           - Live traffic counters per channel")
	fmt.Println("================================================================================")
	fmt.Println(" [✓] Omnichannel Hub active and listening.")

	// Wait for OS termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n [*] Initiating graceful shutdown...")
	cancelCollectors()
	mgr.StopAll()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Stop(shutdownCtx); err != nil {
		log.Printf("[-] Error during shutdown: %v", err)
	}

	fmt.Println(" [✓] Omnichannel Hub safely terminated. Zero data leaks.")
}
