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

// Кэш храним УЖЕ уменьшенным (≤ logoMax px) — иначе сотни полноразмерных
// гербов в RAM приводят к OOM на маленьком инстансе.
const logoMax = 240

var (
	logoCache  = map[string]image.Image{}
	logoInFlt  = map[string]struct{}{}
	logoMu     sync.RWMutex
	logoClient = &http.Client{Timeout: 5 * time.Second}
)

// fetchLogo возвращает герб из кэша. Если его ещё нет — НЕ блокирует запрос:
// запускает фоновую загрузку и сразу возвращает nil (карточка покажет
// инициалы, а на следующем запросе уже будет логотип). Так генерация card.png
// никогда не висит на сетевом запросе → нет таймаутов и 502.
func fetchLogo(url string) image.Image {
	if url == "" {
		return nil
	}
	logoMu.RLock()
	img, ok := logoCache[url]
	logoMu.RUnlock()
	if ok {
		return img // в кэше (может быть nil — негативный результат)
	}

	logoMu.Lock()
	_, cached := logoCache[url]
	_, busy := logoInFlt[url]
	if !cached && !busy {
		logoInFlt[url] = struct{}{}
		go warmLogo(url)
	}
	logoMu.Unlock()
	return nil
}

// warmLogo скачивает, декодирует и УМЕНЬШАЕТ герб, кладёт в кэш (и успех,
// и неудачу как nil, чтобы не долбить мёртвый URL).
func warmLogo(url string) {
	// Паника в фоновой горутине (битый формат картинки и т.п.) НЕ ловится
	// middleware'ом обработчика и роняет ВЕСЬ процесс → 502. Защищаемся.
	defer func() {
		if r := recover(); r != nil {
			logoMu.Lock()
			logoCache[url] = nil // негативный кэш, чтобы не паниковать снова
			logoMu.Unlock()
		}
		logoMu.Lock()
		delete(logoInFlt, url)
		logoMu.Unlock()
	}()

	var decoded image.Image
	if resp, err := logoClient.Get(url); err == nil {
		if resp.StatusCode == http.StatusOK {
			if data, err := io.ReadAll(io.LimitReader(resp.Body, 3<<20)); err == nil {
				if im, _, err := image.Decode(bytes.NewReader(data)); err == nil {
					decoded = scaleDown(im, logoMax)
				}
			}
		}
		resp.Body.Close()
	}

	logoMu.Lock()
	logoCache[url] = decoded
	logoMu.Unlock()
}

// scaleDown уменьшает изображение до maxSide по длинной стороне (если больше),
// возвращая компактный *image.RGBA — чтобы кэш не раздувал память.
func scaleDown(img image.Image, maxSide int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return nil
	}
	longest := w
	if h > w {
		longest = h
	}
	scale := 1.0
	if longest > maxSide {
		scale = float64(maxSide) / float64(longest)
	}
	tw, th := int(float64(w)*scale), int(float64(h)*scale)
	if tw < 1 || th < 1 {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
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
