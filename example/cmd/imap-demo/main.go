// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command imap-demo connects to an IMAP server (default: QQ Mail) and
// fetches recent messages using the ling-base email.IMAPReader.
//
// Usage:
//
//	# Use built-in QQ Mail defaults
//	go run ./cmd/imap-demo
//
//	# Override via env vars
//	IMAP_HOST=imap.gmail.com IMAP_PORT=993 IMAP_USER=you@gmail.com \
//	IMAP_PASS=app-password go run ./cmd/imap-demo
//
//	# Limit number of messages
//	IMAP_LIMIT=10 go run ./cmd/imap-demo
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/LingByte/ling-base/notification/email"
)

func main() {
	host := flag.String("host", envOr("IMAP_HOST", "imap.qq.com"), "IMAP host")
	port := flag.Int("port", envIntOr("IMAP_PORT", 993), "IMAP port")
	user := flag.String("user", envOr("IMAP_USER", "2148582258@qq.com"), "IMAP username")
	pass := flag.String("pass", envOr("IMAP_PASS", "rlpxpibwexfidigf"), "IMAP password / auth code")
	mbox := flag.String("mailbox", envOr("IMAP_MAILBOX", "INBOX"), "mailbox name")
	limit := flag.Int("limit", envIntOr("IMAP_LIMIT", 5), "max messages to fetch")
	flag.Parse()

	cfg := email.IMAPConfig{
		Host:     *host,
		Port:     *port,
		Username: *user,
		Password: *pass,
		Mailbox:  *mbox,
	}

	fmt.Printf("Connecting to IMAP server %s:%d (user=%s, mailbox=%s)...\n",
		cfg.Host, cfg.Port, cfg.Username, cfg.Mailbox)
	reader, err := email.NewIMAPReader(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Connect failed: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()
	fmt.Println("Connected and logged in successfully!")

	fmt.Printf("\nFetching up to %d most recent messages...\n", *limit)
	msgs, err := reader.ReadRecentMessages(*limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fetch failed: %v\n", err)
		os.Exit(1)
	}

	if len(msgs) == 0 {
		fmt.Println("No messages found.")
		return
	}

	fmt.Printf("\nFound %d message(s):\n", len(msgs))
	fmt.Println("========================================")

	for i, msg := range msgs {
		fmt.Printf("\n--- Message %d ---\n", i+1)
		fmt.Printf("ID:       %s\n", msg.ID)
		fmt.Printf("From:     %s <%s>\n", msg.FromName, msg.From)
		fmt.Printf("To:       %v\n", msg.To)
		fmt.Printf("Subject:  %s\n", msg.Subject)
		fmt.Printf("Date:     %s\n", msg.Date.Format("2006-01-02 15:04:05"))
		if msg.ReplyTo != "" {
			fmt.Printf("Reply-To: %s\n", msg.ReplyTo)
		}
		if len(msg.Attachments) > 0 {
			fmt.Printf("Attachments: %d\n", len(msg.Attachments))
			for _, att := range msg.Attachments {
				fmt.Printf("  - %s (%s, %d bytes)\n", att.Filename, att.ContentType, att.Size)
			}
		}
		body := msg.TextBody
		if body == "" {
			body = msg.HTMLBody
		}
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		fmt.Printf("Body:     %s\n", body)
	}

	fmt.Println("\n========================================")
	fmt.Printf("Total: %d message(s)\n", len(msgs))
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
