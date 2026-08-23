//go:build js && wasm

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/LingByte/ling-base/common/barcode"
	"github.com/LingByte/ling-base/common/bloom"
	bloommem "github.com/LingByte/ling-base/common/bloom/memory"
	"github.com/LingByte/ling-base/common/captcha"
	"github.com/LingByte/ling-base/common/compress"
	"github.com/LingByte/ling-base/common/convert"
	"github.com/LingByte/ling-base/common/crypto"
	"github.com/LingByte/ling-base/common/hash"
	"github.com/LingByte/ling-base/common/idgen"
	"github.com/LingByte/ling-base/common/i18n"
	"github.com/LingByte/ling-base/common/jwtutil"
	"github.com/LingByte/ling-base/common/mathutil"
	"github.com/LingByte/ling-base/common/netutil"
	"github.com/LingByte/ling-base/common/nltime"
	"github.com/LingByte/ling-base/common/password"
	"github.com/LingByte/ling-base/common/phone"
	"github.com/LingByte/ling-base/common/pinyin"
	"github.com/LingByte/ling-base/common/qrcode"
	"github.com/LingByte/ling-base/common/random"
	"github.com/LingByte/ling-base/common/totp"
	"github.com/LingByte/ling-base/common/timeutil"
	"github.com/LingByte/ling-base/common/validate"
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

	// ─── Hash ───────────────────────────────────────────
	register("wasmHash", wasmHash)

	// ─── Password ───────────────────────────────────────
	register("wasmPasswordHash", wasmPasswordHash)
	register("wasmPasswordVerify", wasmPasswordVerify)

	// ─── Validate ───────────────────────────────────────
	register("wasmValidate", wasmValidate)

	// ─── JWT ────────────────────────────────────────────
	register("wasmJWTLogin", wasmJWTLogin)
	register("wasmJWTVerify", wasmJWTVerify)

	// ─── QR Code ────────────────────────────────────────
	register("wasmQRCode", wasmQRCode)
	register("wasmQRCodeFancy", wasmQRCodeFancy)

	// ─── Barcode ────────────────────────────────────────
	register("wasmBarcode", wasmBarcode)

	// ─── ID / Random / Pinyin / Phone / Convert / Crypto ─
	register("wasmIDGen", wasmIDGen)
	register("wasmRandom", wasmRandom)
	register("wasmPinyin", wasmPinyin)
	register("wasmPhoneLookup", wasmPhoneLookup)
	register("wasmConvert", wasmConvert)
	register("wasmCryptoAES", wasmCryptoAES)
	register("wasmNLTime", wasmNLTime)
	register("wasmBloomEstimate", wasmBloomEstimate)
	register("wasmBloomDemo", wasmBloomDemo)

	register("wasmCaptchaGenerate", wasmCaptchaGenerate)
	register("wasmCaptchaVerify", wasmCaptchaVerify)
	register("wasmMathStats", wasmMathStats)
	register("wasmNetIP", wasmNetIP)
	register("wasmI18n", wasmI18n)
	register("wasmTimeUtil", wasmTimeUtil)

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

// ─── Hash ───────────────────────────────────────────────

func wasmHash(args []js.Value) any {
	if len(args) < 2 {
		return jsonError(fmt.Errorf("need algo and input"))
	}
	algo := args[0].String()
	input := args[1].String()
	data := []byte(input)

	var out string
	switch algo {
	case "md5":
		out = hash.MD5Hex(data)
	case "sha1":
		out = hash.SHA1Hex(data)
	case "sha256":
		out = hash.SHA256Hex(data)
	case "sha512":
		out = hash.SHA512Hex(data)
	case "hmac-sha256":
		if len(args) < 3 {
			return jsonError(fmt.Errorf("hmac-sha256 needs key"))
		}
		out = hash.HMACSHA256Hex(data, []byte(args[2].String()))
	default:
		return jsonError(fmt.Errorf("unknown algo: %s", algo))
	}
	return jsonResult(map[string]any{"algo": algo, "hex": out})
}

// ─── Password ─────────────────────────────────────────────

func wasmPasswordHash(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need password"))
	}
	plain := args[0].String()
	hashed, err := password.Hash(plain, nil)
	if err != nil {
		return jsonError(err)
	}
	return jsonResult(map[string]any{"hash": hashed})
}

func wasmPasswordVerify(args []js.Value) any {
	if len(args) < 2 {
		return jsonError(fmt.Errorf("need password and hash"))
	}
	plain := args[0].String()
	stored := args[1].String()
	valid := password.Verify(plain, stored)
	return jsonResult(map[string]any{"valid": valid})
}

// ─── Validate ─────────────────────────────────────────────

func wasmValidate(args []js.Value) any {
	if len(args) < 2 {
		return jsonError(fmt.Errorf("need value and tag"))
	}
	value := args[0].String()
	tag := args[1].String()
	err := validate.ValidateWithTag(value, tag)
	if err != nil {
		return jsonResult(map[string]any{"valid": false, "error": err.Error()})
	}
	return jsonResult(map[string]any{"valid": true})
}

// ─── JWT ──────────────────────────────────────────────────

func wasmJWTLogin(args []js.Value) any {
	if len(args) < 2 {
		return jsonError(fmt.Errorf("need secret and subject"))
	}
	secret := args[0].String()
	subject := args[1].String()
	if len(secret) < 32 {
		return jsonError(fmt.Errorf("secret must be at least 32 bytes"))
	}
	auth, err := jwtutil.New(jwtutil.Config{
		Secret:     []byte(secret),
		Issuer:     "ling-base-playground",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
	})
	if err != nil {
		return jsonError(err)
	}
	pair, err := auth.Login(subject)
	if err != nil {
		return jsonError(err)
	}
	return jsonResult(map[string]any{
		"accessToken":  pair.AccessToken,
		"refreshToken": pair.RefreshToken,
		"expiresIn":    pair.ExpiresIn,
	})
}

func wasmJWTVerify(args []js.Value) any {
	if len(args) < 2 {
		return jsonError(fmt.Errorf("need secret and token"))
	}
	secret := args[0].String()
	token := args[1].String()
	auth, err := jwtutil.New(jwtutil.Config{
		Secret:    []byte(secret),
		Issuer:    "ling-base-playground",
		AccessTTL: 15 * time.Minute,
	})
	if err != nil {
		return jsonError(err)
	}
	claims, err := auth.Verify(token)
	if err != nil {
		return jsonResult(map[string]any{"valid": false, "error": err.Error()})
	}
	return jsonResult(map[string]any{
		"valid":   true,
		"subject": claims.Subject,
		"issuer":  claims.Issuer,
		"roles":   claims.Roles,
	})
}

// ─── QR Code ──────────────────────────────────────────────

func wasmQRCode(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need text"))
	}
	text := args[0].String()
	size := 256
	if len(args) > 1 && args[1].Int() > 0 {
		size = args[1].Int()
	}
	png, err := qrcode.GeneratePNG(text, qrcode.ECLMedium, size)
	if err != nil {
		return jsonError(err)
	}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	return jsonResult(map[string]any{
		"size":    size,
		"dataURL": dataURL,
	})
}

type qrFancyRequest struct {
	Module        string `json:"module"`
	Finder        string `json:"finder"`
	FgColor       string `json:"fgColor"`
	BgColor       string `json:"bgColor"`
	BgTransparent bool   `json:"bgTransparent"`
	Level         string `json:"level"`
	ModuleWidth   uint8  `json:"moduleWidth"`
	BorderWidth   int    `json:"borderWidth"`
	Gradient      *struct {
		Angle float64 `json:"angle"`
		Stops []struct {
			T     float64 `json:"t"`
			Color string  `json:"color"`
		} `json:"stops"`
	} `json:"gradient"`
}

func wasmQRCodeFancy(args []js.Value) any {
	if len(args) < 2 {
		return jsonError(fmt.Errorf("need text and options JSON"))
	}
	text := args[0].String()
	var req qrFancyRequest
	if err := json.Unmarshal([]byte(args[1].String()), &req); err != nil {
		return jsonError(fmt.Errorf("invalid options: %w", err))
	}

	opts := qrcode.FancyOptions{
		Module:        parseModuleShape(req.Module),
		Finder:        parseFinderShape(req.Finder),
		FgColor:       parseHexColor(req.FgColor, color.Black),
		BgColor:       parseHexColor(req.BgColor, color.White),
		BgTransparent: req.BgTransparent,
		ModuleWidth:   req.ModuleWidth,
		BorderWidth:   req.BorderWidth,
	}
	if opts.ModuleWidth == 0 {
		opts.ModuleWidth = 21
	}
	if req.Gradient != nil && len(req.Gradient.Stops) >= 2 {
		stops := make([]qrcode.ColorStop, len(req.Gradient.Stops))
		for i, s := range req.Gradient.Stops {
			stops[i] = qrcode.ColorStop{T: s.T, Color: parseHexColor(s.Color, color.Black)}
		}
		opts.FgGradient = qrcode.NewLinearGradient(req.Gradient.Angle, stops...)
	}

	level := parseECLevel(req.Level)
	png, err := qrcode.GenerateFancy(text, level, opts)
	if err != nil {
		return jsonError(err)
	}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	return jsonResult(map[string]any{
		"mode":    "fancy",
		"dataURL": dataURL,
		"options": req,
	})
}

func wasmBarcode(args []js.Value) any {
	if len(args) < 2 {
		return jsonError(fmt.Errorf("need type and content"))
	}
	typ := barcode.BarcodeType(args[0].String())
	content := args[1].String()
	width, height := 300, 100
	if len(args) > 2 && args[2].Int() > 0 {
		width = args[2].Int()
	}
	if len(args) > 3 && args[3].Int() > 0 {
		height = args[3].Int()
	}
	png, err := barcode.GeneratePNG(typ, content, width, height)
	if err != nil {
		return jsonError(err)
	}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	return jsonResult(map[string]any{
		"type":    string(typ),
		"width":   width,
		"height":  height,
		"dataURL": dataURL,
	})
}

func parseModuleShape(s string) qrcode.ModuleShape {
	switch strings.ToLower(s) {
	case "circle":
		return qrcode.ShapeCircle
	case "rounded":
		return qrcode.ShapeRounded
	case "liquid":
		return qrcode.ShapeLiquid
	case "hstripe":
		return qrcode.ShapeHStripe
	case "vstripe":
		return qrcode.ShapeVStripe
	case "diamond":
		return qrcode.ShapeDiamond
	default:
		return qrcode.ShapeRectangle
	}
}

func parseFinderShape(s string) qrcode.FinderShape {
	if strings.ToLower(s) == "rounded" {
		return qrcode.FinderRounded
	}
	return qrcode.FinderSquare
}

func parseECLevel(s string) qrcode.ErrorCorrectionLevel {
	switch strings.ToLower(s) {
	case "low":
		return qrcode.ECLLow
	case "medium":
		return qrcode.ECLMedium
	case "high":
		return qrcode.ECLHigh
	case "quartile":
		return qrcode.ECLQuartile
	default:
		return qrcode.ECLMedium
	}
}

func parseHexColor(s string, fallback color.Color) color.Color {
	s = strings.TrimSpace(strings.TrimPrefix(s, "#"))
	if len(s) != 6 {
		return fallback
	}
	r, e1 := strconv.ParseUint(s[0:2], 16, 8)
	g, e2 := strconv.ParseUint(s[2:4], 16, 8)
	b, e3 := strconv.ParseUint(s[4:6], 16, 8)
	if e1 != nil || e2 != nil || e3 != nil {
		return fallback
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

// ─── ID Gen ───────────────────────────────────────────────

func wasmIDGen(args []js.Value) any {
	kind := "uuidv4"
	if len(args) > 0 && args[0].String() != "" {
		kind = args[0].String()
	}
	switch kind {
	case "uuidv4":
		return jsonResult(map[string]any{"type": kind, "id": idgen.UUIDv4()})
	case "uuidv7":
		return jsonResult(map[string]any{"type": kind, "id": idgen.UUIDv7()})
	case "snowflake":
		return jsonResult(map[string]any{"type": kind, "id": idgen.SnowflakeNext()})
	case "short":
		return jsonResult(map[string]any{"type": kind, "id": idgen.ShortID()})
	case "ordered":
		return jsonResult(map[string]any{"type": kind, "id": idgen.OrderedUUID()})
	default:
		return jsonError(fmt.Errorf("unknown id type: %s", kind))
	}
}

// ─── Random ───────────────────────────────────────────────

func wasmRandom(args []js.Value) any {
	kind := "string"
	n := 16
	if len(args) > 0 && args[0].String() != "" {
		kind = args[0].String()
	}
	if len(args) > 1 && args[1].Int() > 0 {
		n = args[1].Int()
	}
	switch kind {
	case "string":
		return jsonResult(map[string]any{"type": kind, "value": random.String(n)})
	case "numeric":
		return jsonResult(map[string]any{"type": kind, "value": random.NumericString(n)})
	case "hex":
		return jsonResult(map[string]any{"type": kind, "value": random.HexString(n)})
	case "password":
		return jsonResult(map[string]any{"type": kind, "value": random.Password(n)})
	case "uuid":
		return jsonResult(map[string]any{"type": kind, "value": random.UUID()})
	case "color":
		return jsonResult(map[string]any{"type": kind, "value": random.HexColor()})
	case "int":
		min, max := 0, 100
		if len(args) > 1 {
			min = args[1].Int()
		}
		if len(args) > 2 {
			max = args[2].Int()
		}
		return jsonResult(map[string]any{"type": kind, "value": random.IntRange(min, max)})
	default:
		return jsonError(fmt.Errorf("unknown random type: %s", kind))
	}
}

// ─── Pinyin ───────────────────────────────────────────────

func wasmPinyin(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need text"))
	}
	text := args[0].String()
	sep := " "
	if len(args) > 1 {
		sep = args[1].String()
	}
	out := pinyin.Convert(text, pinyin.WithSeparator(sep))
	return jsonResult(map[string]any{"text": text, "pinyin": out})
}

// ─── Phone ────────────────────────────────────────────────

func wasmPhoneLookup(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need phone number"))
	}
	num := args[0].String()
	rec, err := phone.Find(num)
	if err != nil {
		loc := phone.LookupPhoneLocation(num)
		return jsonResult(map[string]any{
			"number":   num,
			"location": loc,
			"error":    err.Error(),
		})
	}
	return jsonResult(map[string]any{
		"number":   num,
		"province": rec.Province,
		"city":     rec.City,
		"cardType": rec.CardType,
		"location": phone.LookupPhoneLocation(num),
	})
}

// ─── Convert ──────────────────────────────────────────────

func wasmConvert(args []js.Value) any {
	if len(args) < 3 {
		return jsonError(fmt.Errorf("need from, to, data"))
	}
	from := convert.Format(args[0].String())
	to := convert.Format(args[1].String())
	data := []byte(args[2].String())
	out, err := convert.Convert(from, to, data)
	if err != nil {
		return jsonError(err)
	}
	return jsonResult(map[string]any{
		"from":   string(from),
		"to":     string(to),
		"result": string(out),
	})
}

// ─── Crypto AES-GCM ───────────────────────────────────────

func wasmCryptoAES(args []js.Value) any {
	if len(args) < 2 {
		return jsonError(fmt.Errorf("need mode and plaintext/ciphertext"))
	}
	mode := args[0].String()
	text := args[1].String()
	keyStr := "0123456789abcdef0123456789abcdef" // 32-byte demo key
	if len(args) > 2 && args[2].String() != "" {
		keyStr = args[2].String()
	}
	key := []byte(keyStr)
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return jsonError(fmt.Errorf("AES key must be 16/24/32 bytes, got %d", len(key)))
	}
	switch mode {
	case "encrypt":
		out, err := crypto.AESGCMEncryptBase64(key, []byte(text))
		if err != nil {
			return jsonError(err)
		}
		return jsonResult(map[string]any{"mode": mode, "ciphertext": out})
	case "decrypt":
		out, err := crypto.AESGCMDecryptBase64(key, text)
		if err != nil {
			return jsonError(err)
		}
		return jsonResult(map[string]any{"mode": mode, "plaintext": string(out)})
	default:
		return jsonError(fmt.Errorf("mode must be encrypt or decrypt"))
	}
}

// ─── Captcha (all types via Manager) ─────────────────────

var playgroundCaptchaMgr = captcha.NewManager(nil)

func wasmCaptchaGenerate(args []js.Value) any {
	typ := captcha.TypeImage
	if len(args) > 0 && args[0].String() != "" {
		typ = captcha.Type(args[0].String())
	}
	res, err := playgroundCaptchaMgr.Generate(typ)
	if err != nil {
		return jsonError(err)
	}
	b, _ := json.Marshal(res)
	var out any
	_ = json.Unmarshal(b, &out)
	return jsonResult(out)
}

type captchaVerifyJSON struct {
	ID    string          `json:"id"`
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

func wasmCaptchaVerify(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need JSON payload"))
	}
	var req captchaVerifyJSON
	if err := json.Unmarshal([]byte(args[0].String()), &req); err != nil {
		return jsonError(err)
	}
	payload := captcha.Payload{ID: req.ID, Type: captcha.Type(req.Type)}
	switch payload.Type {
	case captcha.TypeImage:
		var s string
		if err := json.Unmarshal(req.Value, &s); err != nil {
			return jsonError(err)
		}
		payload.Value = s
	case captcha.TypeClick:
		var pts []captcha.Point
		if err := json.Unmarshal(req.Value, &pts); err != nil {
			return jsonError(err)
		}
		payload.Value = pts
	default:
		var n float64
		if err := json.Unmarshal(req.Value, &n); err != nil {
			return jsonError(err)
		}
		payload.Value = int(n)
	}
	ok, err := playgroundCaptchaMgr.Verify(payload)
	if err != nil {
		return jsonResult(map[string]any{"valid": false, "error": err.Error()})
	}
	return jsonResult(map[string]any{"valid": ok})
}

// ─── Mathutil ─────────────────────────────────────────────

func wasmMathStats(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need JSON number array"))
	}
	var nums []float64
	if err := json.Unmarshal([]byte(args[0].String()), &nums); err != nil {
		return jsonError(err)
	}
	if len(nums) == 0 {
		return jsonError(fmt.Errorf("empty array"))
	}
	return jsonResult(map[string]any{
		"count":   len(nums),
		"sum":     mathutil.Sum(nums),
		"mean":    mathutil.Mean(nums),
		"median":  mathutil.Median(nums),
		"stdDev":  mathutil.StdDev(nums),
		"min":     mathutil.MinSlice(nums),
		"max":     mathutil.MaxSlice(nums),
		"p95":     mathutil.Percentile(nums, 95),
	})
}

// ─── Netutil ──────────────────────────────────────────────

func wasmNetIP(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need IP"))
	}
	ip := args[0].String()
	return jsonResult(map[string]any{
		"ip":        ip,
		"private":   netutil.IsPrivateIP(ip),
		"loopback":  netutil.IsLoopback(ip),
		"public":    netutil.IsPublicIP(ip),
	})
}

// ─── I18n ─────────────────────────────────────────────────

var playgroundI18n = func() *i18n.Manager {
	m := i18n.NewManager(&i18n.Config{
		DefaultLocale:    "zh",
		SupportedLocales: []i18n.Locale{"zh", "en"},
	})
	m.SetTranslation("zh", "welcome", "欢迎使用 ling-base")
	m.SetTranslation("en", "welcome", "Welcome to ling-base")
	m.SetTranslation("zh", "hello", "你好，%s")
	m.SetTranslation("en", "hello", "Hello, %s")
	return m
}()

func wasmI18n(args []js.Value) any {
	if len(args) < 2 {
		return jsonError(fmt.Errorf("need locale and key"))
	}
	locale := i18n.Locale(args[0].String())
	key := args[1].String()
	name := ""
	if len(args) > 2 {
		name = args[2].String()
	}
	if name != "" {
		return jsonResult(map[string]any{"text": playgroundI18n.T(locale, key, name)})
	}
	return jsonResult(map[string]any{"text": playgroundI18n.T(locale, key)})
}

// ─── Timeutil ─────────────────────────────────────────────

func wasmTimeUtil(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need action"))
	}
	action := args[0].String()
	now := time.Now()
	switch action {
	case "format":
		tz := "Asia/Shanghai"
		if len(args) > 1 && args[1].String() != "" {
			tz = args[1].String()
		}
		return jsonResult(map[string]any{
			"now":    now.Format(time.RFC3339),
			"local":  timeutil.FormatIn(now, timeutil.LayoutDateTime, tz),
			"cn":     timeutil.FormatIn(now, timeutil.LayoutCNTime, tz),
		})
	case "startOfDay":
		start := timeutil.StartOfDay(now)
		return jsonResult(map[string]any{
			"start": start.Format(time.RFC3339),
			"end":   timeutil.EndOfDay(now).Format(time.RFC3339),
		})
	case "parse":
		if len(args) < 2 {
			return jsonError(fmt.Errorf("parse needs value"))
		}
		t, err := timeutil.ParseIn(args[1].String(), timeutil.LayoutDateTime, "Asia/Shanghai")
		if err != nil {
			return jsonError(err)
		}
		return jsonResult(map[string]any{"parsed": t.Format(time.RFC3339)})
	default:
		return jsonError(fmt.Errorf("unknown action: %s", action))
	}
}

// ─── NLTime ───────────────────────────────────────────────

func wasmNLTime(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need expression"))
	}
	expr := args[0].String()
	t, err := nltime.Parse(expr, time.Now())
	if err != nil {
		return jsonError(err)
	}
	return jsonResult(map[string]any{
		"expr":   expr,
		"time":   t.Format(time.RFC3339),
		"unix":   t.Unix(),
		"local":  t.Format("2006-01-02 15:04:05"),
	})
}

// ─── Bloom ────────────────────────────────────────────────

func wasmBloomEstimate(args []js.Value) any {
	n := uint64(10000)
	p := 0.01
	if len(args) > 0 && args[0].Int() > 0 {
		n = uint64(args[0].Int())
	}
	if len(args) > 1 {
		p = args[1].Float()
	}
	params, err := bloom.Estimate(n, p)
	if err != nil {
		return jsonError(err)
	}
	return jsonResult(map[string]any{
		"n":    n,
		"p":    p,
		"m":    params.M,
		"k":    params.K,
		"bytes": bloom.BitsToBytes(params.M),
	})
}

func wasmBloomDemo(args []js.Value) any {
	if len(args) < 1 {
		return jsonError(fmt.Errorf("need JSON {add:[], test:[]}"))
	}
	var req struct {
		Add  []string `json:"add"`
		Test []string `json:"test"`
	}
	if err := json.Unmarshal([]byte(args[0].String()), &req); err != nil {
		return jsonError(err)
	}
	params, err := bloom.Estimate(1000, 0.01)
	if err != nil {
		return jsonError(err)
	}
	f, err := bloommem.New(bloommem.WithParams(params))
	if err != nil {
		return jsonError(err)
	}
	ctx := context.Background()
	for _, item := range req.Add {
		_ = f.Add(ctx, item)
	}
	results := map[string]bool{}
	for _, item := range req.Test {
		ok, _ := f.Test(ctx, item)
		results[item] = ok
	}
	return jsonResult(map[string]any{
		"added":  req.Add,
		"tested": results,
		"m":      params.M,
		"k":      params.K,
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
