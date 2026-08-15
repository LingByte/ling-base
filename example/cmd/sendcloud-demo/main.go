// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command sendcloud-demo queries the SendCloud emailStatus API for
// recent delivery records of messages sent via SendCloud, and lists
// configured inbound routes (收信路由).
//
// Usage:
//
//	# Use built-in defaults
//	go run ./cmd/sendcloud-demo
//
//	# Override via env vars
//	SC_API_USER=your_user SC_API_KEY=your_key SC_FROM=you@example.com \
//	go run ./cmd/sendcloud-demo
//
//	# Filter by recipient and limit
//	SC_EMAIL=recipient@example.com SC_LIMIT=20 go run ./cmd/sendcloud-demo
//
//	# List inbound routes only
//	go run ./cmd/sendcloud-demo -routes-only
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/LingByte/ling-base/notification/email"
)

func main() {
	apiUser := flag.String("api-user", envOr("SC_API_USER", "LingEcho"), "SendCloud API user")
	apiKey := flag.String("api-key", envOr("SC_API_KEY", "14b6e48501c452407421917c943be0c3"), "SendCloud API key")
	from := flag.String("from", envOr("SC_FROM", "noreply@email.lingecho.com"), "sender address")
	fromName := flag.String("from-name", envOr("SC_FROM_NAME", "LingEchoX"), "sender display name")
	days := flag.Int("days", envIntOr("SC_DAYS", 3), "query window in days (max 3)")
	emailFilter := flag.String("email", envOr("SC_EMAIL", ""), "filter by recipient address")
	limit := flag.Int("limit", envIntOr("SC_LIMIT", 20), "max records to fetch (max 100)")
	routesOnly := flag.Bool("routes-only", false, "only list inbound routes, skip delivery status")
	flag.Parse()

	cfg := email.SendCloudConfig{
		APIUser:  *apiUser,
		APIKey:   *apiKey,
		From:     *from,
		FromName: *fromName,
	}

	reader, err := email.NewSendCloudReader(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Init SendCloud reader failed: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()

	// ── Delivery status query ──
	if !*routesOnly {
		fmt.Printf("Querying SendCloud delivery status (apiUser=%s, days=%d, email=%q, limit=%d)...\n",
			cfg.APIUser, *days, *emailFilter, *limit)

		q := email.SendCloudStatusQuery{
			Days:  *days,
			Limit: *limit,
			Email: *emailFilter,
		}

		records, total, err := reader.QueryStatus(q)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nAPI reported total: %d, returned: %d record(s)\n", total, len(records))
		fmt.Println("========================================")

		for i, rec := range records {
			fmt.Printf("\n--- Record %d ---\n", i+1)
			fmt.Printf("EmailID:     %s\n", rec.EmailID)
			fmt.Printf("Recipient:   %s\n", rec.Recipients)
			fmt.Printf("Status:      %s\n", rec.Status)
			if rec.SubStatus != "" {
				fmt.Printf("SubStatus:   %s (%s)\n", rec.SubStatus, rec.SubStatusDesc)
			}
			fmt.Printf("APIUser:     %s\n", rec.APIUser)
			fmt.Printf("RequestTime: %s\n", rec.RequestTime)
			fmt.Printf("ModifiedTime:%s\n", rec.ModifiedTime)
			if rec.SendLog != "" {
				fmt.Printf("SendLog:     %s\n", rec.SendLog)
			}
			fmt.Printf("MailStatus:  %s\n", email.SendCloudStatusToMailStatus(rec.Status))
		}

		fmt.Println("\n========================================")
		fmt.Printf("Total: %d record(s)\n", len(records))
	}

	// ── Inbound routes ──
	fmt.Println("\n--- Inbound Routes (收信路由) ---")
	routes, err := reader.ListInboundRoutes("", 0, 100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "List inbound routes failed: %v\n", err)
	} else if len(routes) == 0 {
		fmt.Println("No inbound routes configured.")
	} else {
		fmt.Printf("Found %d route(s):\n", len(routes))
		for i, rt := range routes {
			fmt.Printf("  [%d] id=%d domain=%s expression=%s action=%s",
				i+1, rt.ID, rt.Domain, rt.Expression, rt.Action)
			if rt.APIUserRoute != "" {
				fmt.Printf(" apiUserRoute=%s", rt.APIUserRoute)
			}
			fmt.Println()
		}
	}

	if *routesOnly {
		return
	}

	// ── MailReader interface demo ──
	fmt.Println("\n--- Via MailReader interface (ReadMessages) ---")
	msgs, err := reader.ReadMessages(*limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ReadMessages failed: %v\n", err)
		os.Exit(1)
	}
	if len(msgs) == 0 {
		fmt.Println("No delivery records found.")
	} else {
		for i, msg := range msgs {
			fmt.Printf("[%d] id=%s from=%s to=%v subject=%s\n",
				i+1, msg.ID, msg.From, msg.To, msg.Subject)
		}
	}
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
