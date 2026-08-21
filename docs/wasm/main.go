//go:build js && wasm

// ling-base WASM playground bridge.
// 编译: GOOS=js GOARCH=wasm go build -o public/lingbase.wasm main.go
// 前端通过 wasmexec.js 加载后，调用注册的 JS 函数执行各模块能力。
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/LingByte/ling-base/common/compress"
	"github.com/LingByte/ling-base/common/password"
	"github.com/LingByte/ling-base/common/totp"
)

func main() {
	register := func(name string, fn func(this js.Value, args []js.Value) any) {
		js.Global().Set(name, js.FuncOf(fn))
	}

	// ─── TOTP ───────────────────────────────────────────
	register("wasmTOTPGenerate", wasmTOTPGenerate)
	register("wasmTOTPValidate", wasmTOTPValidate)
	register("wasmTOTPCurrentCode", wasmTOTPCurrentCode)
	register("wasmTOTPQRDataURL", wasmTOTPQRDataURL)
	register("wasmTOTPBackupCodes", wasmTOTPBackupCodes)

	// ─── Password ───────────────────────────────────────
	register("wasmPasswordHash", wasmPasswordHash)
	register("wasmPasswordVerify", wasmPasswordVerify)
	register("wasmPasswordNeedsRehash", wasmPasswordNeedsRehash)

	// ─── Compress ───────────────────────────────────────
	register("wasmZstdCompress", wasmZstdCompress)
	register("wasmZstdDecompress", wasmZstdDecompress)
	register("wasmSnappyCompress", wasmSnappyCompress)
	register("wasmSnappyDecompress", wasmSnappyDecompress)
	register("wasmLZ4Compress", wasmLZ4Compress)
	register("wasmLZ4Decompress", wasmLZ4Decompress)
	register("wasmGzipCompress", wasmGzipCompress)
	register("wasmGzipDecompress", wasmGzipDecompress)

	// 通知 JS 层已就绪
	js.Global().Set("lingbaseWasmReady", js.ValueOf(true))

	// 保持进程运行 — 用 channel 而非 select{} 避免 WASM 退出
	done := make(chan struct{})
	<-done
}

// ─── TOTP ───────────────────────────────────────────────

func wasmTOTPGenerate(this js.Value, args []js.Value) any {
	issuer := "ling-base"
	account := "user@example.com"
	if len(args) > 0 && args[0].Length() > 0 {
		issuer = args[0].String()
	}
	if len(args) > 1 && args[1].Length() > 0 {
		account = args[1].String()
	}

	key, err := totp.Generate(totp.Options{
		Issuer:      issuer,
		AccountName: account,
	})
	if err != nil {
		return errorJSON(err)
	}

	qrURL, _ := totp.QRDataURL(key.URL(), 256)

	result := map[string]any{
		"secret":    key.Secret(),
		"url":       key.URL(),
		"qrDataUrl": qrURL,
	}
	return toJSON(result)
}

func wasmTOTPValidate(this js.Value, args []js.Value) any {
	if len(args) < 2 {
		return errorJSON(fmt.Errorf("need code and secret"))
	}
	code := args[0].String()
	secret := args[1].String()
	valid := totp.Validate(code, secret, nil)
	return toJSON(map[string]any{"valid": valid})
}

func wasmTOTPCurrentCode(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON(fmt.Errorf("need secret"))
	}
	secret := args[0].String()
	code, err := totp.CurrentCode(secret, nil)
	if err != nil {
		return errorJSON(err)
	}
	return toJSON(map[string]any{"code": code, "expiresIn": 30 - time.Now().Second()%30})
}

func wasmTOTPQRDataURL(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON(fmt.Errorf("need otpauth url"))
	}
	url := args[0].String()
	size := 256
	if len(args) > 1 && args[1].Int() > 0 {
		size = args[1].Int()
	}
	dataURL, err := totp.QRDataURL(url, size)
	if err != nil {
		return errorJSON(err)
	}
	return toJSON(map[string]any{"qrDataUrl": dataURL})
}

func wasmTOTPBackupCodes(this js.Value, args []js.Value) any {
	count := 10
	if len(args) > 0 && args[0].Int() > 0 {
		count = args[0].Int()
	}
	codes, hashes, err := totp.GenerateBackupCodes(totp.BackupOptions{
		Count:     count,
		Length:    8,
		Separator: "-",
	})
	if err != nil {
		return errorJSON(err)
	}
	return toJSON(map[string]any{"codes": codes, "hashes": hashes})
}

// ─── Password ───────────────────────────────────────────

func wasmPasswordHash(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON(fmt.Errorf("need password"))
	}
	plain := args[0].String()
	algorithm := "argon2id"
	if len(args) > 1 && args[1].Length() > 0 {
		algorithm = args[1].String()
	}

	opts := &password.Options{Algorithm: password.AlgorithmArgon2id}
	switch algorithm {
	case "bcrypt":
		opts.Algorithm = password.AlgorithmBcrypt
	case "argon2id":
		opts.Algorithm = password.AlgorithmArgon2id
	}

	hash, err := password.Hash(plain, opts)
	if err != nil {
		return errorJSON(err)
	}
	return toJSON(map[string]any{"hash": hash, "algorithm": algorithm})
}

func wasmPasswordVerify(this js.Value, args []js.Value) any {
	if len(args) < 2 {
		return errorJSON(fmt.Errorf("need password and hash"))
	}
	plain := args[0].String()
	stored := args[1].String()
	valid := password.Verify(plain, stored)
	return toJSON(map[string]any{"valid": valid})
}

func wasmPasswordNeedsRehash(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON(fmt.Errorf("need hash"))
	}
	stored := args[0].String()
	needs := password.NeedsRehash(stored, nil)
	return toJSON(map[string]any{"needsRehash": needs})
}

// ─── Compress ───────────────────────────────────────────

func wasmZstdCompress(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	compressed, err := compress.ZstdCompress(data)
	if err != nil {
		return errorJSON(err)
	}
	return compressResult(data, compressed)
}

func wasmZstdDecompress(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	decompressed, err := compress.ZstdDecompress(data)
	if err != nil {
		return errorJSON(err)
	}
	return toJSON(map[string]any{
		"originalSize": len(data),
		"resultSize":   len(decompressed),
		"result":       string(decompressed),
	})
}

func wasmSnappyCompress(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	compressed, err := compress.SnappyCompress(data)
	if err != nil {
		return errorJSON(err)
	}
	return compressResult(data, compressed)
}

func wasmSnappyDecompress(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	decompressed, err := compress.SnappyDecompress(data)
	if err != nil {
		return errorJSON(err)
	}
	return toJSON(map[string]any{
		"originalSize": len(data),
		"resultSize":   len(decompressed),
		"result":       string(decompressed),
	})
}

func wasmLZ4Compress(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	compressed, err := compress.LZ4Compress(data)
	if err != nil {
		return errorJSON(err)
	}
	return compressResult(data, compressed)
}

func wasmLZ4Decompress(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	decompressed, err := compress.LZ4Decompress(data)
	if err != nil {
		return errorJSON(err)
	}
	return toJSON(map[string]any{
		"originalSize": len(data),
		"resultSize":   len(decompressed),
		"result":       string(decompressed),
	})
}

func wasmGzipCompress(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	compressed, err := compress.GzipCompressDefault(data)
	if err != nil {
		return errorJSON(err)
	}
	return compressResult(data, compressed)
}

func wasmGzipDecompress(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	decompressed, err := compress.GzipDecompress(data)
	if err != nil {
		return errorJSON(err)
	}
	return toJSON(map[string]any{
		"originalSize": len(data),
		"resultSize":   len(decompressed),
		"result":       string(decompressed),
	})
}

// ─── Helpers ────────────────────────────────────────────

func compressResult(original, compressed []byte) any {
	ratio := 0.0
	if len(original) > 0 {
		ratio = float64(len(compressed)) / float64(len(original)) * 100
	}
	return toJSON(map[string]any{
		"originalSize":  len(original),
		"compressedSize": len(compressed),
		"ratio":         fmt.Sprintf("%.1f%%", ratio),
		"compressedB64": base64.StdEncoding.EncodeToString(compressed),
	})
}

func jsToBytes(v js.Value) []byte {
	length := v.Length()
	out := make([]byte, length)
	for i := 0; i < length; i++ {
		out[i] = byte(v.Index(i).Int())
	}
	return out
}

func toJSON(v any) js.Value {
	b, _ := json.Marshal(v)
	return js.ValueOf(string(b))
}

func errorJSON(err error) js.Value {
	b, _ := json.Marshal(map[string]any{"error": err.Error()})
	return js.ValueOf(string(b))
}
