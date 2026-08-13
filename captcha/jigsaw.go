package captcha

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"time"
)

const (
	defaultJigsawWidth     = 300
	defaultJigsawHeight    = 150
	defaultJigsawPieceSize = 40
	defaultJigsawTolerance = 5
)

// jigsawStored holds the target X offset for the puzzle piece.
type jigsawStored struct {
	TargetX int
}

// JigsawCaptcha is a slider puzzle challenge: a piece is cut from the image
// and the user must drag it back to the correct horizontal position.
type JigsawCaptcha struct {
	width      int
	height     int
	pieceSize  int
	tolerance  int
	expiration time.Duration
	store      Store
}

// NewJigsawCaptcha creates a jigsaw captcha generator.
func NewJigsawCaptcha(width, height, pieceSize, tolerance int, expiration time.Duration, store Store) *JigsawCaptcha {
	if store == nil {
		store = NewMemoryStore()
	}
	if width <= 0 {
		width = defaultJigsawWidth
	}
	if height <= 0 {
		height = defaultJigsawHeight
	}
	if pieceSize <= 0 {
		pieceSize = defaultJigsawPieceSize
	}
	if tolerance < 0 {
		tolerance = defaultJigsawTolerance
	}
	return &JigsawCaptcha{
		width:      width,
		height:     height,
		pieceSize:  pieceSize,
		tolerance:  tolerance,
		expiration: expiration,
		store:      store,
	}
}

// Generate creates a jigsaw challenge. The response includes the background
// image (with the piece area masked) and the puzzle piece image, both as PNG
// data URLs. The user drags the piece horizontally to fill the gap.
func (jc *JigsawCaptcha) Generate() (*Result, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	bg, piece, targetX, err := jc.generateImages(rng)
	if err != nil {
		return nil, fmt.Errorf("failed to generate jigsaw images: %w", err)
	}

	bgURL, err := encodePNGDataURL(bg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode background: %w", err)
	}
	pieceURL, err := encodePNGDataURL(piece)
	if err != nil {
		return nil, fmt.Errorf("failed to encode piece: %w", err)
	}

	id := generateID()
	expires := time.Now().Add(jc.expiration)
	if err := jc.store.Set(id, jigsawStored{TargetX: targetX}, expires); err != nil {
		return nil, fmt.Errorf("failed to store captcha: %w", err)
	}

	return &Result{
		ID:   id,
		Type: TypeJigsaw,
		Data: map[string]interface{}{
			"width":      jc.width,
			"height":     jc.height,
			"pieceSize":  jc.pieceSize,
			"background": bgURL,
			"piece":      pieceURL,
			"tolerance":  jc.tolerance,
		},
		Expires: expires,
	}, nil
}

// Verify checks that the user dragged the piece to the correct X position
// within the tolerance margin.
func (jc *JigsawCaptcha) Verify(id string, userX int) (bool, error) {
	return jc.store.VerifyWithFunc(id, userX, jc.compare)
}

// compare checks whether the user's X position matches the target within tolerance.
func (jc *JigsawCaptcha) compare(stored, input interface{}) bool {
	s, ok1 := stored.(jigsawStored)
	v, ok2 := input.(int)
	if !ok1 || !ok2 {
		return false
	}
	return abs(v-s.TargetX) <= jc.tolerance
}

// generateImages creates the background (with masked cutout) and the puzzle
// piece image. Returns the target X position where the piece should be placed.
func (jc *JigsawCaptcha) generateImages(rng *rand.Rand) (bg, piece image.Image, targetX int, err error) {
	// Generate a random gradient background.
	bgImg := image.NewRGBA(image.Rect(0, 0, jc.width, jc.height))
	top := color.RGBA{
		uint8(rng.Intn(100) + 80),
		uint8(rng.Intn(100) + 80),
		uint8(rng.Intn(100) + 120),
		255,
	}
	bottom := color.RGBA{
		uint8(rng.Intn(80) + 40),
		uint8(rng.Intn(80) + 100),
		uint8(rng.Intn(80) + 140),
		255,
	}
	for y := 0; y < jc.height; y++ {
		t := float64(y) / float64(max(1, jc.height-1))
		for x := 0; x < jc.width; x++ {
			r := clampByte(int(float64(top.R)*(1-t)+float64(bottom.R)*t), 0)
			g := clampByte(int(float64(top.G)*(1-t)+float64(bottom.G)*t), 0)
			b := clampByte(int(float64(top.B)*(1-t)+float64(bottom.B)*t), 0)
			bgImg.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Add interference lines.
	for i := 0; i < 8; i++ {
		x1 := rng.Intn(jc.width)
		y1 := rng.Intn(jc.height)
		x2 := rng.Intn(jc.width)
		y2 := rng.Intn(jc.height)
		drawLine(bgImg, x1, y1, x2, y2, color.RGBA{
			uint8(rng.Intn(100) + 80),
			uint8(rng.Intn(100) + 80),
			uint8(rng.Intn(100) + 80),
			60,
		})
	}

	// Choose a target X position for the cutout (right half of the image,
	// leaving room for the piece).
	minX := jc.pieceSize + 10
	maxX := jc.width - jc.pieceSize - 10
	if maxX <= minX {
		maxX = minX + 1
	}
	targetX = rng.Intn(maxX-minX) + minX

	// The piece Y position is vertically centered.
	pieceY := (jc.height - jc.pieceSize) / 2

	// Extract the puzzle piece and mask the background.
	pieceImg := image.NewRGBA(image.Rect(0, 0, jc.pieceSize, jc.pieceSize))
	for py := 0; py < jc.pieceSize; py++ {
		for px := 0; px < jc.pieceSize; px++ {
			srcX := targetX + px
			srcY := pieceY + py
			if srcX >= 0 && srcX < jc.width && srcY >= 0 && srcY < jc.height {
				c := bgImg.RGBAAt(srcX, srcY)
				pieceImg.Set(px, py, c)
				// Mask the background with a dark overlay.
				bgImg.Set(srcX, srcY, color.RGBA{c.R / 3, c.G / 3, c.B / 3, 255})
			}
		}
	}

	// Draw a border around the piece for visibility.
	border := color.RGBA{255, 255, 255, 200}
	for i := 0; i < jc.pieceSize; i++ {
		pieceImg.Set(i, 0, border)
		pieceImg.Set(i, jc.pieceSize-1, border)
		pieceImg.Set(0, i, border)
		pieceImg.Set(jc.pieceSize-1, i, border)
	}

	return bgImg, pieceImg, targetX, nil
}

// encodePNGDataURL encodes an image as a PNG data URL.
func encodePNGDataURL(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
