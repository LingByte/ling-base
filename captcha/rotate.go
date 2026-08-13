package captcha

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"time"
)

const (
	defaultRotateSize      = 200
	defaultRotateTolerance = 15 // degrees
)

// rotateStored holds the target rotation angle (degrees, 0-359).
type rotateStored struct {
	Angle int
}

// RotateCaptcha is a rotation challenge: an image is rotated by a random
// angle and the user must rotate it back to upright (0 degrees).
// The server stores the angle the image was rotated by; the user submits the
// angle they rotated it back. A pass requires the residual to be within
// tolerance of 0 (mod 360).
type RotateCaptcha struct {
	size       int
	tolerance  int // tolerance in degrees
	expiration time.Duration
	store      Store
}

// NewRotateCaptcha creates a rotate captcha generator.
func NewRotateCaptcha(size, tolerance int, expiration time.Duration, store Store) *RotateCaptcha {
	if store == nil {
		store = NewMemoryStore()
	}
	if size <= 0 {
		size = defaultRotateSize
	}
	if tolerance < 0 {
		tolerance = defaultRotateTolerance
	}
	return &RotateCaptcha{
		size:       size,
		tolerance:  tolerance,
		expiration: expiration,
		store:      store,
	}
}

// Generate creates a rotate challenge. The response includes a circular image
// rotated by a random angle; the user must rotate it back to upright.
func (rc *RotateCaptcha) Generate() (*Result, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	angle := rng.Intn(360)

	img := rc.generateImage(rng)
	rotated := rc.rotate(img, float64(angle))

	imgURL, err := encodePNGDataURL(rotated)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	id := generateID()
	expires := time.Now().Add(rc.expiration)
	if err := rc.store.Set(id, rotateStored{Angle: angle}, expires); err != nil {
		return nil, fmt.Errorf("failed to store captcha: %w", err)
	}

	return &Result{
		ID:   id,
		Type: TypeRotate,
		Data: map[string]interface{}{
			"size":      rc.size,
			"image":     imgURL,
			"tolerance": rc.tolerance,
		},
		Expires: expires,
	}, nil
}

// Verify checks that the user's rotation angle brings the image within
// tolerance of upright. The user submits the angle they rotated; the residual
// is (storedAngle - userAngle) mod 360, which must be within tolerance of 0.
func (rc *RotateCaptcha) Verify(id string, userAngle int) (bool, error) {
	return rc.store.VerifyWithFunc(id, userAngle, rc.compare)
}

// compare checks whether the residual rotation is within tolerance of 0 degrees.
func (rc *RotateCaptcha) compare(stored, input interface{}) bool {
	s, ok1 := stored.(rotateStored)
	v, ok2 := input.(int)
	if !ok1 || !ok2 {
		return false
	}
	residual := (s.Angle - v) % 360
	if residual < 0 {
		residual += 360
	}
	return residual <= rc.tolerance || residual >= 360-rc.tolerance
}

// generateImage creates a simple circular image with a distinctive top marker
// so the user can tell which way is up.
func (rc *RotateCaptcha) generateImage(rng *rand.Rand) image.Image {
	size := rc.size
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	center := float64(size) / 2
	radius := float64(size)/2 - 2

	// Fill with a gradient circle.
	top := color.RGBA{
		uint8(rng.Intn(80) + 100),
		uint8(rng.Intn(80) + 100),
		uint8(rng.Intn(80) + 150),
		255,
	}
	bottom := color.RGBA{
		uint8(rng.Intn(60) + 60),
		uint8(rng.Intn(60) + 120),
		uint8(rng.Intn(60) + 140),
		255,
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - center
			dy := float64(y) - center
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= radius {
				t := (float64(y) + 1) / float64(size)
				r := clampByte(int(float64(top.R)*(1-t)+float64(bottom.R)*t), 0)
				g := clampByte(int(float64(top.G)*(1-t)+float64(bottom.G)*t), 0)
				b := clampByte(int(float64(top.B)*(1-t)+float64(bottom.B)*t), 0)
				img.Set(x, y, color.RGBA{r, g, b, 255})
			} else {
				img.Set(x, y, color.RGBA{240, 240, 240, 255})
			}
		}
	}

	// Draw a distinctive arrow/triangle at the top so the user can identify
	// the correct orientation.
	arrowColor := color.RGBA{255, 80, 80, 255}
	arrowHeight := size / 5
	arrowWidth := size / 8
	cx := size / 2
	for dy := 0; dy < arrowHeight; dy++ {
		halfW := arrowWidth * (arrowHeight - dy) / arrowHeight / 2
		for dx := -halfW; dx <= halfW; dx++ {
			px := cx + dx
			py := 4 + dy
			if px >= 0 && px < size && py >= 0 && py < size {
				img.Set(px, py, arrowColor)
			}
		}
	}

	// Add a few decorative dots for visual texture.
	for i := 0; i < 20; i++ {
		px := rng.Intn(size)
		py := rng.Intn(size)
		dx := float64(px) - center
		dy := float64(py) - center
		if dx*dx+dy*dy <= radius*radius {
			img.Set(px, py, color.RGBA{
				uint8(rng.Intn(100) + 100),
				uint8(rng.Intn(100) + 100),
				uint8(rng.Intn(100) + 100),
				255,
			})
		}
	}

	return img
}

// rotate rotates img by angle degrees clockwise and returns a new image of
// the same dimensions. Pixels outside the original image bounds become
// transparent.
func (rc *RotateCaptcha) rotate(img image.Image, angle float64) image.Image {
	size := rc.size
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	center := float64(size) / 2

	rad := angle * math.Pi / 180
	cos := math.Cos(rad)
	sin := math.Sin(rad)

	// Inverse rotation: for each destination pixel, find the source pixel.
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - center
			dy := float64(y) - center
			// Inverse rotation matrix.
			sx := dx*cos + dy*sin + center
			sy := -dx*sin + dy*cos + center
			if sx >= 0 && sx < float64(size) && sy >= 0 && sy < float64(size) {
				dst.Set(x, y, img.At(int(sx), int(sy)))
			} else {
				dst.Set(x, y, color.RGBA{240, 240, 240, 255})
			}
		}
	}

	return dst
}
