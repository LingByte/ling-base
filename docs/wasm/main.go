//go:build js && wasm

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/LingByte/ling-base/common/compress"
	"github.com/LingByte/ling-base/common/totp"
)

func safeCall(fn func(args []js.Value) any) func(this js.Value, args []js.Value) any {
	return func(this js.Value, args []js.Value) (result any) {
		defer func() {
			if r := recover(); r != nil {
				b, _ := json.Marshal(map[string]any{"error": fmt.Sprintf("panic: %v", r)})
				result = js.ValueOf(string(b))
			}
		}()
		result = fn(args)
		return
	}
}

func register(name string, fn func(args []js.Value) any) {
	js.Global().Set(name, js.FuncOf(safeCall(fn)))
}

func main() {
	// ─── TOTP（不依赖 crypto/rand 的功能）──────────────
	register("wasmTOTPValidate", wasmTOTPValidate)
	register("wasmTOTPCurrentCode", wasmTOTPCurrentCode)
	register("wasmTOTPBackupCodes", wasmTOTPBackupCodes)

	// ─── Compress ───────────────────────────────────────
	register("wasmZstdCompress", wasmZstdCompress)
	register("wasmZstdDecompress", wasmZstdDecompress)
	register("wasmSnappyCompress", wasmSnappyCompress)
	register("wasmSnappyDecompress", wasmSnappyDecompress)
	register("wasmLZ4Compress", wasmLZ4Compress)
	register("wasmLZ4Decompress", wasmLZ4Decompress)
	register("wasmGzipCompress", wasmGzipCompress)
	register("wasmGzipDecompress", wasmGzipDecompress)

	js.Global().Set("lingbaseWasmReady", js.ValueOf(true))

	select {}
}

// ─── TOTP ───────────────────────────────────────────────

func wasmTOTPValidate(args []js.Value) any {
	if len(args) < 2 {
		return jsonError(fmt.Errorf("need code and secret"))
	}
	code := args[0].String()
	secret := args[1].String()
	valid := totp.Validate(code, secret, nil)
	return jsonResult(map[string]any{"valid": valid})
}

func wasmTOTPCurrentCode(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need secret"))
	}
	secret := args[0].String()
	code, err := totp.CurrentCode(secret, nil)
	if err != nil {
		return jsonError(err)
	}
	return jsonResult(map[string]any{"code": code, "expiresIn": 30 - time.Now().Second()%30})
}

func wasmTOTPBackupCodes(args []js.Value) any {
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
		return jsonError(err)
	}
	return jsonResult(map[string]any{"codes": codes, "hashes": hashes})
}

// ─── Compress ───────────────────────────────────────────

func wasmZstdCompress(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	compressed, err := compress.ZstdCompress(data)
	if err != nil {
		return jsonError(err)
	}
	return compressResult(data, compressed)
}

func wasmZstdDecompress(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	decompressed, err := compress.ZstdDecompress(data)
	if err != nil {
		return jsonError(err)
	}
	return jsonResult(map[string]any{
		"originalSize": len(data),
		"resultSize":   len(decompressed),
		"result":       string(decompressed),
	})
}

func wasmSnappyCompress(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	compressed, err := compress.SnappyCompress(data)
	if err != nil {
		return jsonError(err)
	}
	return compressResult(data, compressed)
}

func wasmSnappyDecompress(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	decompressed, err := compress.SnappyDecompress(data)
	if err != nil {
		return jsonError(err)
	}
	return jsonResult(map[string]any{
		"originalSize": len(data),
		"resultSize":   len(decompressed),
		"result":       string(decompressed),
	})
}

func wasmLZ4Compress(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	compressed, err := compress.LZ4Compress(data)
	if err != nil {
		return jsonError(err)
	}
	return compressResult(data, compressed)
}

func wasmLZ4Decompress(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	decompressed, err := compress.LZ4Decompress(data)
	if err != nil {
		return jsonError(err)
	}
	return jsonResult(map[string]any{
		"originalSize": len(data),
		"resultSize":   len(decompressed),
		"result":       string(decompressed),
	})
}

func wasmGzipCompress(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	compressed, err := compress.GzipCompressDefault(data)
	if err != nil {
		return jsonError(err)
	}
	return compressResult(data, compressed)
}

func wasmGzipDecompress(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need data"))
	}
	data := jsToBytes(args[0])
	decompressed, err := compress.GzipDecompress(data)
	if err != nil {
		return jsonError(err)
	}
	return jsonResult(map[string]any{
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
	return jsonResult(map[string]any{
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

func jsonResult(v any) js.Value {
	b, _ := json.Marshal(v)
	return js.ValueOf(string(b))
}

func jsonError(err error) js.Value {
	b, _ := json.Marshal(map[string]any{"error": err.Error()})
	return js.ValueOf(string(b))
}
