package parser

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeOCRProvider is a test double for OCRProvider.
type fakeOCRProvider struct {
	name      string
	text      string
	err       error
	lastImage []byte
	lastOpts  *OCROptions
}

func (f *fakeOCRProvider) Name() string { return f.name }
func (f *fakeOCRProvider) Recognize(ctx context.Context, imageBytes []byte, opts *OCROptions) (string, error) {
	f.lastImage = imageBytes
	f.lastOpts = opts
	if f.err != nil {
		return "", f.err
	}
	return f.text, nil
}

// validPNG1x1 is a minimal 1x1 transparent PNG used for testing.
var validPNG1x1 = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4,
	0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44,
	0xae, 0x42, 0x60, 0x82,
}

func TestOCRParser_NoProviderRegistered(t *testing.T) {
	// Save and restore global provider.
	saved := GetOCRProvider()
	SetOCRProvider(nil)
	defer SetOCRProvider(saved)

	p := &OCRParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypePNG,
		FileName: "test.png",
		Content:  validPNG1x1,
	}, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedFileType))
}

func TestOCRParser_WithFakeProvider(t *testing.T) {
	fp := &fakeOCRProvider{name: "fake", text: "hello world"}

	saved := GetOCRProvider()
	SetOCRProvider(fp)
	defer SetOCRProvider(saved)

	p := &OCRParser{Language: "en"}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypePNG,
		FileName: "test.png",
		Content:  validPNG1x1,
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Equal(t, "hello world", res.Text)
	assert.Equal(t, FileTypePNG, res.FileType)
	assert.Equal(t, "test.png", res.FileName)
	assert.Len(t, res.Sections, 1)
	assert.Equal(t, "test.png", res.Sections[0].Title)

	// Verify the provider received the image bytes and options.
	assert.Equal(t, validPNG1x1, fp.lastImage)
	assert.Equal(t, "en", fp.lastOpts.Language)
}

func TestOCRParser_InstanceDriverOverridesGlobal(t *testing.T) {
	globalFP := &fakeOCRProvider{name: "global", text: "global text"}
	instanceFP := &fakeOCRProvider{name: "instance", text: "instance text"}

	saved := GetOCRProvider()
	SetOCRProvider(globalFP)
	defer SetOCRProvider(saved)

	p := &OCRParser{Driver: instanceFP}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypePNG,
		FileName: "test.png",
		Content:  validPNG1x1,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "instance text", res.Text)

	// Global provider should not have been called.
	assert.Nil(t, globalFP.lastImage)
	assert.NotNil(t, instanceFP.lastImage)
}

func TestOCRParser_ProviderError(t *testing.T) {
	fp := &fakeOCRProvider{name: "fake", err: errors.New("api timeout")}

	saved := GetOCRProvider()
	SetOCRProvider(fp)
	defer SetOCRProvider(saved)

	p := &OCRParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypePNG,
		FileName: "test.png",
		Content:  validPNG1x1,
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api timeout")
	assert.Contains(t, err.Error(), "fake")
}

func TestOCRParser_InvalidImage(t *testing.T) {
	fp := &fakeOCRProvider{name: "fake", text: "hello"}

	saved := GetOCRProvider()
	SetOCRProvider(fp)
	defer SetOCRProvider(saved)

	p := &OCRParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypePNG,
		FileName: "test.png",
		Content:  []byte("not an image"),
	}, nil)
	require.Error(t, err)
	// Should fail at image.Decode, not reach the provider.
	assert.Nil(t, fp.lastImage)
}

func TestOCRParser_EmptyInput(t *testing.T) {
	fp := &fakeOCRProvider{name: "fake", text: "hello"}

	saved := GetOCRProvider()
	SetOCRProvider(fp)
	defer SetOCRProvider(saved)

	p := &OCRParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypePNG,
		FileName: "test.png",
		Content:  []byte{},
	}, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyInput))
}

func TestOCRParser_NilRequest(t *testing.T) {
	p := &OCRParser{}
	_, err := p.Parse(context.Background(), nil, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyInput))
}

func TestOCRParser_SupportedTypes(t *testing.T) {
	p := &OCRParser{}
	types := p.SupportedTypes()
	assert.Contains(t, types, FileTypePNG)
	assert.Contains(t, types, FileTypeJPG)
	assert.Contains(t, types, FileTypeJPEG)
	assert.Contains(t, types, FileTypeWEBP)
	assert.Contains(t, types, FileTypeGIF)
	assert.Contains(t, types, FileTypeBMP)
	assert.Contains(t, types, FileTypeTIFF)
	assert.Contains(t, types, FileTypeTIF)
}

func TestOCRParser_ProviderName(t *testing.T) {
	p := &OCRParser{}
	assert.Equal(t, "ocr", p.Provider())
}

func TestRegisterOCRProvider(t *testing.T) {
	fp1 := &fakeOCRProvider{name: "fake1"}
	fp2 := &fakeOCRProvider{name: "fake2"}

	RegisterOCRProvider("fake1", fp1)
	RegisterOCRProvider("fake2", fp2)

	drivers := RegisteredOCRDrivers()
	assert.Contains(t, drivers, "fake1")
	assert.Contains(t, drivers, "fake2")
}

func TestSetOCRProviderByDriver(t *testing.T) {
	fp := &fakeOCRProvider{name: "test-driver"}
	RegisterOCRProvider("test-driver", fp)

	saved := GetOCRProvider()
	defer SetOCRProvider(saved)

	err := SetOCRProviderByDriver("test-driver")
	require.NoError(t, err)
	assert.Equal(t, fp, GetOCRProvider())

	// Unknown driver should error.
	err = SetOCRProviderByDriver("nonexistent")
	require.Error(t, err)
}

func TestSetOCRProviderByDriver_NilProvider(t *testing.T) {
	RegisterOCRProvider("nil-test", nil)
	drivers := RegisteredOCRDrivers()
	assert.NotContains(t, drivers, "nil-test")
}

func TestRouter_Parse_OCRRequiresProvider(t *testing.T) {
	saved := GetOCRProvider()
	SetOCRProvider(nil)
	defer SetOCRProvider(saved)

	r := DefaultRouter()
	_, err := r.Parse(context.Background(), &ParseRequest{
		FileName: "test.png",
		Content:  validPNG1x1,
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OCR")
}

func TestRouter_Parse_OCRWithProvider(t *testing.T) {
	fp := &fakeOCRProvider{name: "fake", text: "routed text"}

	saved := GetOCRProvider()
	SetOCRProvider(fp)
	defer SetOCRProvider(saved)

	r := DefaultRouter()
	res, err := r.Parse(context.Background(), &ParseRequest{
		FileName: "test.png",
		Content:  validPNG1x1,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "routed text", res.Text)
}
