package cardgen

import (
	"bytes"
	_ "embed"
	"image/color"
	"strconv"
	"strings"
	"unicode"

	clubdata "efootball-bot/internal/data"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

//go:embed fonts/LiberationSans-Bold.ttf
var fontBold []byte

//go:embed fonts/LiberationSans-Regular.ttf
var fontRegular []byte

type CardData struct {
	DisplayName  string
	Rank         string
	Rating       int
	TeamPower    int
	FavoriteClub string
	Wins         int
	Draws        int
	Losses       int
	TotalGoals   int
	WinRate      float64
	Achievements []string
}

// GenerateCard рисует премиальную карточку игрока, брендированную под цвета
// клуба. Без сетевых запросов — полностью локальный рендер.
func GenerateCard(d CardData) ([]byte, error) {
	const s = 1.7
	W, H := int(600*s), int(350*s)
	dc := gg.NewContext(W, H)

	club := findClub(d.FavoriteClub)
	accent := accentColor(club, d.Rating)
	accent2 := secondColor(club, accent)

	// ── Фон: тёмный градиент ──
	bg := gg.NewLinearGradient(0, 0, float64(W), float64(H))
	bg.AddColorStop(0, color.RGBA{24, 28, 41, 255})
	bg.AddColorStop(1, color.RGBA{7, 10, 17, 255})
	dc.SetFillStyle(bg)
	dc.DrawRectangle(0, 0, float64(W), float64(H))
	dc.Fill()

	// ── Свечение цвета клуба (NRGBA — не premultiplied) ──
	glow := gg.NewRadialGradient(s*120, s*64, 0, s*120, s*64, s*330)
	glow.AddColorStop(0, color.NRGBA{accent.R, accent.G, accent.B, 66})
	glow.AddColorStop(1, color.NRGBA{accent.R, accent.G, accent.B, 0})
	dc.SetFillStyle(glow)
	dc.DrawRectangle(0, 0, float64(W), float64(H))
	dc.Fill()

	// ── Диагональный блик (премиум-ощущение) ──
	dc.SetColor(color.NRGBA{255, 255, 255, 7})
	dc.MoveTo(float64(W)*0.56, 0)
	dc.LineTo(float64(W)*0.70, 0)
	dc.LineTo(float64(W)*0.40, float64(H))
	dc.LineTo(float64(W)*0.26, float64(H))
	dc.ClosePath()
	dc.Fill()

	// ── Декоративная гигантская инициала ──
	_ = loadFont(dc, fontBold, s*200)
	dc.SetRGBA(float64(accent.R)/255, float64(accent.G)/255, float64(accent.B)/255, 0.06)
	dc.DrawStringAnchored(getInitials(d.DisplayName), s*480, s*220, 0.5, 0.5)

	// ── Фирменный бейдж клуба (кружок-щит с монограммой) ──
	cx, cy, rr := s*106, s*150, s*68.0
	ring := gg.NewLinearGradient(cx-rr, cy-rr, cx+rr, cy+rr)
	ring.AddColorStop(0, accent)
	ring.AddColorStop(1, accent2)
	dc.SetFillStyle(ring)
	dc.DrawCircle(cx, cy, rr+s*5)
	dc.Fill()

	dc.DrawCircle(cx, cy, rr)
	dc.Clip()
	core := gg.NewLinearGradient(cx-rr, cy-rr, cx+rr, cy+rr)
	core.AddColorStop(0, accent)
	core.AddColorStop(1, accent2)
	dc.SetFillStyle(core)
	dc.DrawRectangle(cx-rr, cy-rr, 2*rr, 2*rr)
	dc.Fill()
	dc.ResetClip()

	// внутреннее светлое кольцо — глубина
	dc.SetColor(color.NRGBA{255, 255, 255, 45})
	dc.SetLineWidth(s * 1.5)
	dc.DrawCircle(cx, cy, rr-s*6)
	dc.Stroke()

	// монограмма
	_ = loadFont(dc, fontBold, s*48)
	dc.SetColor(contrastText(accent))
	dc.DrawStringAnchored(getInitials(d.DisplayName), cx, cy, 0.5, 0.42)

	// название клуба
	if club != nil {
		_ = loadFont(dc, fontBold, s*15)
		dc.SetRGBA(1, 1, 1, 0.92)
		dc.DrawStringAnchored(truncate(clubName(club), 18), cx, cy+rr+s*30, 0.5, 0.5)
	}

	// ── Правая колонка ──
	rx := s * 206

	// бейдж ранга
	rank := upperCase(cleanRank(d.Rank))
	if rank != "" {
		drawPill(dc, s, rx, s*34, rank, accent)
	}

	// имя
	_ = loadFont(dc, fontBold, s*30)
	dc.SetRGBA(1, 1, 1, 0.98)
	dc.DrawString(truncate(d.DisplayName, 18), rx, s*96)

	// ELO + лейбл
	elo := strconv.Itoa(d.Rating)
	_ = loadFont(dc, fontBold, s*56)
	dc.SetColor(accent)
	dc.DrawString(elo, rx, s*158)
	ew, _ := dc.MeasureString(elo)
	_ = loadFont(dc, fontRegular, s*15)
	dc.SetRGBA(1, 1, 1, 0.4)
	dc.DrawString("ELO", rx+ew+s*12, s*152)

	// ── Панель статистики ──
	py := s * 206
	pw := float64(W) - rx - s*22
	dc.SetColor(color.NRGBA{255, 255, 255, 10})
	dc.DrawRoundedRectangle(rx, py, pw, s*92, s*14)
	dc.Fill()
	dc.SetColor(color.NRGBA{255, 255, 255, 20})
	dc.SetLineWidth(s * 1)
	dc.DrawRoundedRectangle(rx, py, pw, s*92, s*14)
	dc.Stroke()

	sy := py + s*46
	step := pw / 5
	cx0 := rx + step/2
	drawStat(dc, s, cx0, sy, strconv.Itoa(d.Wins), "W", color.RGBA{34, 197, 94, 255})
	drawStat(dc, s, cx0+step, sy, strconv.Itoa(d.Draws), "D", color.RGBA{234, 179, 8, 255})
	drawStat(dc, s, cx0+2*step, sy, strconv.Itoa(d.Losses), "L", color.RGBA{239, 68, 68, 255})
	drawStat(dc, s, cx0+3*step, sy, formatPct(d.WinRate), "WIN", color.RGBA{129, 140, 248, 255})
	drawStat(dc, s, cx0+4*step, sy, strconv.Itoa(d.TotalGoals), "GOALS", color.RGBA{251, 146, 60, 255})

	// ── Тонкая акцентная рамка ──
	dc.SetColor(color.NRGBA{accent.R, accent.G, accent.B, 95})
	dc.SetLineWidth(s * 1.5)
	dc.DrawRoundedRectangle(s*8, s*8, float64(W)-s*16, float64(H)-s*16, s*18)
	dc.Stroke()

	// ── Нижняя акцентная полоса ──
	stripe := gg.NewLinearGradient(0, 0, float64(W), 0)
	stripe.AddColorStop(0, accent)
	stripe.AddColorStop(1, accent2)
	dc.SetFillStyle(stripe)
	dc.DrawRectangle(0, float64(H)-s*5, float64(W), s*5)
	dc.Fill()

	// ── Футер ──
	_ = loadFont(dc, fontBold, s*12)
	dc.SetRGBA(1, 1, 1, 0.32)
	dc.DrawStringAnchored("eFootLeague", float64(W)-s*22, float64(H)-s*24, 1, 0.5)

	var buf bytes.Buffer
	if err := encodePNGStdlib(&buf, dc.Image()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawPill(dc *gg.Context, s, x, y float64, text string, accent color.RGBA) {
	_ = loadFont(dc, fontBold, s*12)
	tw, _ := dc.MeasureString(text)
	padX := s * 11
	h := s * 24
	w := tw + 2*padX
	dc.SetColor(color.NRGBA{accent.R, accent.G, accent.B, 32})
	dc.DrawRoundedRectangle(x, y, w, h, h/2)
	dc.Fill()
	dc.SetColor(accent)
	dc.SetLineWidth(s * 1.4)
	dc.DrawRoundedRectangle(x, y, w, h, h/2)
	dc.Stroke()
	dc.SetColor(accent)
	dc.DrawStringAnchored(text, x+w/2, y+h/2, 0.5, 0.42)
}

func drawStat(dc *gg.Context, s, x, y float64, val, label string, col color.Color) {
	_ = loadFont(dc, fontBold, s*20)
	dc.SetColor(col)
	dc.DrawStringAnchored(val, x, y, 0.5, 0.5)
	_ = loadFont(dc, fontRegular, s*10)
	dc.SetRGBA(1, 1, 1, 0.42)
	dc.DrawStringAnchored(label, x, y+s*18, 0.5, 0.5)
}

func formatPct(v float64) string { return strconv.Itoa(int(v+0.5)) + "%" }

// ── Клуб и цвета ──────────────────────────────────────────────────────────────

func findClub(id string) *clubdata.Club {
	if id == "" {
		return nil
	}
	for i := range clubdata.Clubs {
		if clubdata.Clubs[i].ID == id {
			return &clubdata.Clubs[i]
		}
	}
	return nil
}

func clubName(c *clubdata.Club) string {
	if c.NameRu != "" {
		return c.NameRu
	}
	return c.Name
}

func hexColor(s string) color.RGBA {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return color.RGBA{120, 120, 130, 255}
	}
	r, _ := strconv.ParseUint(s[0:2], 16, 8)
	g, _ := strconv.ParseUint(s[2:4], 16, 8)
	b, _ := strconv.ParseUint(s[4:6], 16, 8)
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

func luminance(c color.RGBA) float64 {
	return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
}

func accentColor(club *clubdata.Club, rating int) color.RGBA {
	if club != nil {
		if c := hexColor(club.Color); luminance(c) >= 45 {
			return c
		}
		if c := hexColor(club.Color2); luminance(c) >= 45 {
			return c
		}
	}
	return tierAccent(rating)
}

func secondColor(club *clubdata.Club, accent color.RGBA) color.RGBA {
	if club != nil {
		if c2 := hexColor(club.Color2); colorDist(c2, accent) > 60 {
			return c2
		}
		if c1 := hexColor(club.Color); colorDist(c1, accent) > 60 {
			return c1
		}
	}
	return darken(accent, 0.55)
}

func colorDist(a, b color.RGBA) float64 {
	dr := float64(a.R) - float64(b.R)
	dg := float64(a.G) - float64(b.G)
	db := float64(a.B) - float64(b.B)
	return dr*dr + dg*dg + db*db
}

func darken(c color.RGBA, f float64) color.RGBA {
	return color.RGBA{uint8(float64(c.R) * f), uint8(float64(c.G) * f), uint8(float64(c.B) * f), 255}
}

func contrastText(c color.RGBA) color.Color {
	if luminance(c) > 150 {
		return color.RGBA{15, 18, 25, 255}
	}
	return color.White
}

func tierAccent(rating int) color.RGBA {
	switch {
	case rating >= 1500:
		return color.RGBA{251, 191, 36, 255}
	case rating >= 1300:
		return color.RGBA{96, 165, 250, 255}
	case rating >= 1150:
		return color.RGBA{167, 139, 250, 255}
	default:
		return color.RGBA{74, 222, 128, 255}
	}
}

// ── Текст ─────────────────────────────────────────────────────────────────────

func cleanRank(s string) string {
	runes := []rune(s)
	i := 0
	for i < len(runes) && !unicode.IsLetter(runes[i]) {
		i++
	}
	return strings.TrimSpace(string(runes[i:]))
}

func upperCase(s string) string { return strings.ToUpper(s) }

func getInitials(name string) string {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "?"
	}
	out := string([]rune(parts[0])[0:1])
	if len(parts) > 1 {
		out += string([]rune(parts[1])[0:1])
	}
	return strings.ToUpper(out)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

func loadFont(dc *gg.Context, data []byte, pts float64) error {
	face, err := parseFontFace(data, pts)
	if err != nil {
		return err
	}
	dc.SetFontFace(face)
	return nil
}

func parseFontFace(data []byte, pts float64) (font.Face, error) {
	f, err := truetype.Parse(data)
	if err != nil {
		return nil, err
	}
	return truetype.NewFace(f, &truetype.Options{Size: pts}), nil
}
