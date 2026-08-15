package captcha

import (
	"crypto/rand"
	"encoding/binary"
)

// LoginCaptchaTypes are the kinds randomly issued for auth flows.
// Slider and click are excluded because they are inconvenient on mobile H5 browsers.
var LoginCaptchaTypes = []Type{TypeImage, TypeMath, TypeJigsaw, TypeRotate}

// RandomType picks one captcha kind uniformly at random.
func RandomType() Type {
	types := LoginCaptchaTypes
	if len(types) == 0 {
		return TypeSlider
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return types[0]
	}
	n := binary.BigEndian.Uint64(b[:])
	return types[int(n%uint64(len(types)))]
}
