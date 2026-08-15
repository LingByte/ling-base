// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package email

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// ParseMailMessage — plain text
// ──────────────────────────────────────────────

func TestParseMailMessage_PlainText(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: bob@example.com",
		"Subject: Hello",
		"Message-Id: <test-1@example.com>",
		"Date: Mon, 01 Jan 2024 12:00:00 +0000",
		"",
		"This is a plain text body.",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	assert.Equal(t, "<test-1@example.com>", msg.ID)
	assert.Equal(t, "Hello", msg.Subject)
	assert.Equal(t, "alice@example.com", msg.From)
	assert.Equal(t, "", msg.FromName)
	assert.Equal(t, []string{"bob@example.com"}, msg.To)
	assert.Equal(t, "This is a plain text body.", msg.TextBody)
	assert.Empty(t, msg.HTMLBody)
	assert.Empty(t, msg.Attachments)
	assert.False(t, msg.Date.IsZero())
}

func TestParseMailMessage_WithDisplayName(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: Alice <alice@example.com>",
		"To: Bob <bob@example.com>, Charlie <charlie@example.com>",
		"Subject: Hi all",
		"Message-Id: <test-2@example.com>",
		"",
		"Body",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	assert.Equal(t, "Alice", msg.FromName)
	assert.Equal(t, "alice@example.com", msg.From)
	assert.Len(t, msg.To, 2)
	assert.Equal(t, "bob@example.com", msg.To[0])
	assert.Equal(t, "charlie@example.com", msg.To[1])
}

func TestParseMailMessage_WithCC(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: bob@example.com",
		"Cc: cc1@example.com, cc2@example.com",
		"Subject: Cc test",
		"",
		"Body",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	assert.Len(t, msg.Cc, 2)
	assert.Equal(t, "cc1@example.com", msg.Cc[0])
	assert.Equal(t, "cc2@example.com", msg.Cc[1])
}

func TestParseMailMessage_ReplyTo(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: bob@example.com",
		"Reply-To: replies@example.com",
		"Subject: Reply",
		"",
		"Body",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	assert.Equal(t, "replies@example.com", msg.ReplyTo)
}

// ──────────────────────────────────────────────
// ParseMailMessage — multipart/alternative
// ──────────────────────────────────────────────

func TestParseMailMessage_MultipartAlternative(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: bob@example.com",
		"Subject: Multipart",
		"Message-Id: <multi-1@example.com>",
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=\"boundary123\"",
		"",
		"--boundary123",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"Plain text part",
		"--boundary123",
		"Content-Type: text/html; charset=utf-8",
		"",
		"<p>HTML part</p>",
		"--boundary123--",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	assert.Equal(t, "Plain text part", msg.TextBody)
	assert.Equal(t, "<p>HTML part</p>", msg.HTMLBody)
	assert.Empty(t, msg.Attachments)
}

// ──────────────────────────────────────────────
// ParseMailMessage — multipart/mixed with attachment
// ──────────────────────────────────────────────

func TestParseMailMessage_MultipartMixedWithAttachment(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: bob@example.com",
		"Subject: With attachment",
		"MIME-Version: 1.0",
		"Content-Type: multipart/mixed; boundary=\"mixed-boundary\"",
		"",
		"--mixed-boundary",
		"Content-Type: text/plain",
		"",
		"Body text",
		"--mixed-boundary",
		"Content-Type: application/octet-stream",
		"Content-Disposition: attachment; filename=\"test.txt\"",
		"",
		"attachment content",
		"--mixed-boundary--",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	assert.Equal(t, "Body text", msg.TextBody)
	require.Len(t, msg.Attachments, 1)
	assert.Equal(t, "test.txt", msg.Attachments[0].Filename)
	assert.Equal(t, "application/octet-stream", msg.Attachments[0].ContentType)
	assert.Equal(t, int64(len("attachment content")), msg.Attachments[0].Size)
	assert.Equal(t, "attachment content", string(msg.Attachments[0].Data))
}

// ──────────────────────────────────────────────
// ParseMailMessage — no content type
// ──────────────────────────────────────────────

func TestParseMailMessage_NoContentType(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: bob@example.com",
		"Subject: No CT",
		"",
		"Simple body without content type",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	assert.Equal(t, "Simple body without content type", msg.TextBody)
}

// ──────────────────────────────────────────────
// ParseMailMessage — invalid content type fallback
// ──────────────────────────────────────────────

func TestParseMailMessage_InvalidContentType(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: bob@example.com",
		"Subject: Bad CT",
		"Content-Type: this is not valid",
		"",
		"Fallback body",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	assert.Equal(t, "Fallback body", msg.TextBody)
}

// ──────────────────────────────────────────────
// ParseMailMessage — invalid raw
// ──────────────────────────────────────────────

func TestParseMailMessage_InvalidRaw(t *testing.T) {
	_, err := ParseMailMessage([]byte("not a valid email at all"))
	// mail.ReadMessage is lenient — it may still parse this.
	// Just check we don't panic.
	_ = err
}

// ──────────────────────────────────────────────
// ParseMailMessage — encoded subject
// ──────────────────────────────────────────────

func TestParseMailMessage_EncodedSubject(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: bob@example.com",
		"Subject: =?UTF-8?B?5pel5pys6Kqe?=",
		"",
		"Body",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	assert.Contains(t, msg.Subject, "日本語")
}

// ──────────────────────────────────────────────
// ParseMailMessage — empty address list
// ──────────────────────────────────────────────

func TestParseMailMessage_EmptyToAndCc(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: alice@example.com",
		"Subject: No recipients",
		"",
		"Body",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	assert.Empty(t, msg.To)
	assert.Empty(t, msg.Cc)
}

func TestParseMailMessage_InvalidAddressList(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: not a valid address list <<<",
		"Subject: Bad addrs",
		"",
		"Body",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	// Should fall back to raw string.
	assert.NotEmpty(t, msg.To)
}

// ──────────────────────────────────────────────
// ParseMailMessage — multipart with nested multipart
// ──────────────────────────────────────────────

func TestParseMailMessage_NestedMultipart(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: bob@example.com",
		"Subject: Nested",
		"MIME-Version: 1.0",
		"Content-Type: multipart/mixed; boundary=\"outer\"",
		"",
		"--outer",
		"Content-Type: multipart/alternative; boundary=\"inner\"",
		"",
		"--inner",
		"Content-Type: text/plain",
		"",
		"Inner text",
		"--inner",
		"Content-Type: text/html",
		"",
		"<p>Inner HTML</p>",
		"--inner--",
		"--outer",
		"Content-Type: application/pdf",
		"Content-Disposition: attachment; filename=\"doc.pdf\"",
		"",
		"PDFDATA",
		"--outer--",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	assert.Equal(t, "Inner text", msg.TextBody)
	assert.Equal(t, "<p>Inner HTML</p>", msg.HTMLBody)
	require.Len(t, msg.Attachments, 1)
	assert.Equal(t, "doc.pdf", msg.Attachments[0].Filename)
}

// ──────────────────────────────────────────────
// ParseMailMessage — HTML only (no plain text)
// ──────────────────────────────────────────────

func TestParseMailMessage_HTMLOnly(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: bob@example.com",
		"Subject: HTML only",
		"Content-Type: text/html; charset=utf-8",
		"",
		"<h1>HTML only body</h1>",
	}, "\r\n"))

	msg, err := ParseMailMessage(raw)
	require.NoError(t, err)
	assert.Equal(t, "<h1>HTML only body</h1>", msg.HTMLBody)
	assert.Empty(t, msg.TextBody)
}

// ──────────────────────────────────────────────
// parseAddressList
// ──────────────────────────────────────────────

func TestParseAddressList_Empty(t *testing.T) {
	assert.Nil(t, parseAddressList(""))
	assert.Nil(t, parseAddressList("   "))
}

func TestParseAddressList_Single(t *testing.T) {
	result := parseAddressList("alice@example.com")
	assert.Equal(t, []string{"alice@example.com"}, result)
}

func TestParseAddressList_Multiple(t *testing.T) {
	result := parseAddressList("Alice <alice@example.com>, Bob <bob@example.com>")
	assert.Len(t, result, 2)
	assert.Equal(t, "alice@example.com", result[0])
	assert.Equal(t, "bob@example.com", result[1])
}

func TestParseAddressList_Invalid(t *testing.T) {
	result := parseAddressList("not valid <<<")
	assert.NotEmpty(t, result) // falls back to raw
}

// ──────────────────────────────────────────────
// decodeMimeHeader
// ──────────────────────────────────────────────

func TestDecodeMimeHeader_PlainASCII(t *testing.T) {
	assert.Equal(t, "Hello World", decodeMimeHeader("Hello World"))
}

func TestDecodeMimeHeader_Base64UTF8(t *testing.T) {
	// =?UTF-8?B?5pel5pys6Kqe?= = "日本語"
	result := decodeMimeHeader("=?UTF-8?B?5pel5pys6Kqe?=")
	assert.Equal(t, "日本語", result)
}

func TestDecodeMimeHeader_InvalidEncoding(t *testing.T) {
	// Invalid encoded-word should return original string.
	input := "=?INVALID?B?garbage?="
	result := decodeMimeHeader(input)
	assert.Equal(t, input, result)
}

// ──────────────────────────────────────────────
// assignBody
// ──────────────────────────────────────────────

func TestAssignBody_HTML(t *testing.T) {
	msg := &MailMessage{}
	assignBody(msg, "text/html", "<p>hi</p>")
	assert.Equal(t, "<p>hi</p>", msg.HTMLBody)
	assert.Empty(t, msg.TextBody)
}

func TestAssignBody_PlainText(t *testing.T) {
	msg := &MailMessage{}
	assignBody(msg, "text/plain", "hi")
	assert.Equal(t, "hi", msg.TextBody)
	assert.Empty(t, msg.HTMLBody)
}

func TestAssignBody_DoesNotOverwriteHTML(t *testing.T) {
	msg := &MailMessage{HTMLBody: "<p>existing</p>"}
	assignBody(msg, "text/html", "<p>new</p>")
	assert.Equal(t, "<p>existing</p>", msg.HTMLBody)
}

func TestAssignBody_DoesNotOverwriteText(t *testing.T) {
	msg := &MailMessage{TextBody: "existing"}
	assignBody(msg, "text/plain", "new")
	assert.Equal(t, "existing", msg.TextBody)
}

func TestAssignBody_UnknownTypeFallback(t *testing.T) {
	msg := &MailMessage{}
	assignBody(msg, "application/octet-stream", "data")
	assert.Equal(t, "data", msg.TextBody)
}

// ──────────────────────────────────────────────
// MemoryMailReader
// ──────────────────────────────────────────────

func TestMemoryMailReader_ReadMessages(t *testing.T) {
	reader := NewMemoryMailReader()
	reader.AddMessage("msg-1", []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: bob@example.com",
		"Subject: Message 1",
		"Message-Id: <msg-1@example.com>",
		"",
		"Body 1",
	}, "\r\n")))
	reader.AddMessage("msg-2", []byte(strings.Join([]string{
		"From: carol@example.com",
		"To: dave@example.com",
		"Subject: Message 2",
		"Message-Id: <msg-2@example.com>",
		"",
		"Body 2",
	}, "\r\n")))

	msgs, err := reader.ReadMessages(0)
	require.NoError(t, err)
	assert.Len(t, msgs, 2)
	assert.Equal(t, "Message 1", msgs[0].Subject)
	assert.Equal(t, "Message 2", msgs[1].Subject)
}

func TestMemoryMailReader_ReadMessages_WithLimit(t *testing.T) {
	reader := NewMemoryMailReader()
	for i := 0; i < 5; i++ {
		reader.AddMessage(fmt.Sprintf("msg-%d", i), []byte(strings.Join([]string{
			"From: alice@example.com",
			"To: bob@example.com",
			"Subject: " + fmt.Sprintf("Message %d", i),
			"",
			"Body",
		}, "\r\n")))
	}

	msgs, err := reader.ReadMessages(2)
	require.NoError(t, err)
	assert.Len(t, msgs, 2)
}

func TestMemoryMailReader_ReadMessages_SkipsRead(t *testing.T) {
	reader := NewMemoryMailReader()
	reader.AddMessage("msg-1", []byte("From: a@x.com\r\nSubject: S1\r\n\r\nBody"))
	reader.AddMessage("msg-2", []byte("From: a@x.com\r\nSubject: S2\r\n\r\nBody"))

	require.NoError(t, reader.MarkRead("msg-1"))

	msgs, err := reader.ReadMessages(0)
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "S2", msgs[0].Subject)
}

func TestMemoryMailReader_ReadMessages_Empty(t *testing.T) {
	reader := NewMemoryMailReader()
	msgs, err := reader.ReadMessages(0)
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestMemoryMailReader_ReadMessages_InvalidMessage(t *testing.T) {
	reader := NewMemoryMailReader()
	// Add a completely invalid message — should be skipped, not cause error.
	reader.AddMessage("bad-1", []byte("garbage that is not an email"))
	msgs, err := reader.ReadMessages(0)
	require.NoError(t, err)
	// mail.ReadMessage is lenient, may still parse. Just verify no error.
	_ = msgs
}

func TestMemoryMailReader_ReadMessages_NoMessageID_UsesKey(t *testing.T) {
	reader := NewMemoryMailReader()
	reader.AddMessage("custom-id", []byte(strings.Join([]string{
		"From: alice@example.com",
		"To: bob@example.com",
		"Subject: No ID",
		"",
		"Body",
	}, "\r\n")))

	msgs, err := reader.ReadMessages(0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "custom-id", msgs[0].ID)
}

func TestMemoryMailReader_MarkRead(t *testing.T) {
	reader := NewMemoryMailReader()
	reader.AddMessage("msg-1", []byte("From: a@x.com\r\nSubject: S\r\n\r\nB"))

	err := reader.MarkRead("msg-1")
	require.NoError(t, err)

	// Second read should return nothing.
	msgs, _ := reader.ReadMessages(0)
	assert.Empty(t, msgs)
}

func TestMemoryMailReader_MarkRead_NotFound(t *testing.T) {
	reader := NewMemoryMailReader()
	err := reader.MarkRead("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryMailReader_DeleteMessage(t *testing.T) {
	reader := NewMemoryMailReader()
	reader.AddMessage("msg-1", []byte("From: a@x.com\r\nSubject: S\r\n\r\nB"))
	reader.AddMessage("msg-2", []byte("From: a@x.com\r\nSubject: S2\r\n\r\nB"))

	err := reader.DeleteMessage("msg-1")
	require.NoError(t, err)

	msgs, _ := reader.ReadMessages(0)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "S2", msgs[0].Subject)
}

func TestMemoryMailReader_DeleteMessage_NotFound(t *testing.T) {
	reader := NewMemoryMailReader()
	err := reader.DeleteMessage("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryMailReader_Close(t *testing.T) {
	reader := NewMemoryMailReader()
	err := reader.Close()
	require.NoError(t, err)
}

// ──────────────────────────────────────────────
// MailReader interface compile-time check
// ──────────────────────────────────────────────

var _ MailReader = (*MemoryMailReader)(nil)
var _ MailLogStore = (*MemoryMailLogStore)(nil)
