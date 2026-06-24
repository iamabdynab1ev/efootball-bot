package cardgen

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"sync"
	"time"

	xdraw "golang.org/x/image/draw"

	"github.com/fogleman/gg"
)

var (
	logoCache  = map[string]image.Image{}
	logoMu     sync.RWMutex
	logoClient = &http.Client{Timeout: 4 * time.Second}
)

// fetchLogo скачивает и декодирует герб клуба (PNG/JPEG) с кэшем в памяти.
// Возвращает nil при любой ошибке — вызывающий рисует фолбэк.
func fetchLogo(url string) image.Image {
	if url == "" {
		return nil
	}
	logoMu.RLock()
	img, ok := logoCache[url]
	logoMu.RUnlock()
	if ok {
		return img // может быть nil — негативный кэш
	}

	var decoded image.Image
	if resp, err := logoClient.Get(url); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			if data, err := io.ReadAll(io.LimitReader(resp.Body, 3<<20)); err == nil {
				if im, _, err := image.Decode(bytes.NewReader(data)); err == nil {
					decoded = im
				}
			}
		}
	}

	logoMu.Lock()
	logoCache[url] = decoded // кэшируем и успех, и неудачу (nil)
	logoMu.Unlock()
	return decoded
}

// drawLogoFit рисует изображение по центру (cx,cy), вписывая в квадрат box,
// сохраняя пропорции, со сглаживанием.
func drawLogoFit(dc *gg.Context, img image.Image, cx, cy, box float64) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return
	}
	longest := w
	if h > w {
		longest = h
	}
	scale := box / float64(longest)
	tw, th := int(float64(w)*scale), int(float64(h)*scale)
	if tw < 1 || th < 1 {
		return
	}
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	dc.DrawImageAnchored(dst, int(cx), int(cy), 0.5, 0.5)
}
