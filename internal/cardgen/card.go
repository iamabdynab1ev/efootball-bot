package cardgen

import (
	"bytes"
	_ "embed"
	"fmt"
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

// GenerateCard рисует премиальную карточку игрока (брендирована под цвета клуба).
// Рендер в 2x для чёткости при шаринге.
func GenerateCard(d CardData) ([]byte, error) {
	const s = 2.0
	W, H := int(600*s), int(350*s)
	dc := gg.NewContext(W, H)

	club := findClub(d.FavoriteClub)
	accent := accentColor(club, d.Rating)
	accent2 := secondColor(club, accent)

	// ── Фон: тёмный градиент ──
	bgGrad := gg.NewLinearGradient(0, 0, 0, float64(H))
	bgGrad.AddColorStop(0, color.RGBA{20, 24, 36, 255})
	bgGrad.AddColorStop(1, color.RGBA{6, 9, 16, 255})
	dc.SetFillStyle(bgGrad)
	dc.DrawRectangle(0, 0, float64(W), float64(H))
	dc.Fill()

	// ── Свечение цвета клуба сверху-слева ──
	// color.NRGBA (не premultiplied) — иначе альфа заливает всю карточку цветом.
	glow := gg.NewRadialGradient(s*120, s*70, 0, s*120, s*70, s*300)
	glow.AddColorStop(0, color.NRGBA{accent.R, accent.G, accent.B, 70})
	glow.AddColorStop(1, color.NRGBA{accent.R, accent.G, accent.B, 0})
	dc.SetFillStyle(glow)
	dc.DrawRectangle(0, 0, float64(W), float64(H))
	dc.Fill()

	// ── Декоративная гигантская инициала (еле видна, справа) ──
	_ = loadFont(dc, fontBold, s*300)
	dc.SetRGBA(float64(accent.R)/255, float64(accent.G)/255, float64(accent.B)/255, 0.06)
	dc.DrawStringAnchored(getInitials(d.DisplayName), s*475, s*210, 0.5, 0.5)

	// ── Крест-кружок с градиентом клуба + кольцо ──
	cx, cy, rr := s*108, s*148, s*70.0
	dc.SetRGBA(1, 1, 1, 0.14)
	dc.DrawCircle(cx, cy, rr+s*4)
	dc.Fill()
	dc.DrawCircle(cx, cy, rr)
	dc.Clip()
	cg := gg.NewLinearGradient(cx-rr, cy-rr, cx+rr, cy+rr)
	cg.AddColorStop(0, accent)
	cg.AddColorStop(1, accent2)
	dc.SetFillStyle(cg)
	dc.DrawRectangle(cx-rr, cy-rr, 2*rr, 2*rr)
	dc.Fill()
	dc.ResetClip()

	// Инициалы игрока на кресте
	_ = loadFont(dc, fontBold, s*50)
	dc.SetColor(contrastText(accent))
	dc.DrawStringAnchored(getInitials(d.DisplayName), cx, cy, 0.5, 0.42)

	// Название клуба под крестом
	if club != nil {
		_ = loadFont(dc, fontBold, s*16)
		dc.SetRGBA(1, 1, 1, 0.92)
		dc.DrawStringAnchored(truncate(clubName(club), 18), cx, cy+rr+s*28, 0.5, 0.5)
	}

	// ── Правая колонка ──
	rx := s * 210

	// Имя
	_ = loadFont(dc, fontBold, s*30)
	dc.SetRGBA(1, 1, 1, 0.98)
	dc.DrawString(truncate(d.DisplayName, 18), rx, s*70)

	// Ранг (чистим эмодзи, в верхнем регистре)
	rank := upperCase(cleanRank(d.Rank))
	if rank != "" {
		_ = loadFont(dc, fontRegular, s*14)
		dc.SetRGBA(1, 1, 1, 0.5)
		dc.DrawString(rank, rx, s*96)
	}

	// ELO + лейбл
	elo := strconv.Itoa(d.Rating)
	_ = loadFont(dc, fontBold, s*54)
	dc.SetColor(accent)
	dc.DrawString(elo, rx, s*156)
	ew, _ := dc.MeasureString(elo)
	_ = loadFont(dc, fontRegular, s*14)
	dc.SetRGBA(1, 1, 1, 0.4)
	dc.DrawString("ELO", rx+ew+s*12, s*150)

	// ── Панель статистики ──
	py := s * 210
	dc.SetRGBA(1, 1, 1, 0.04)
	dc.DrawRoundedRectangle(rx, py, float64(W)-rx-s*24, s*86, s*12)
	dc.Fill()
	dc.SetRGBA(1, 1, 1, 0.07)
	dc.SetLineWidth(s * 1)
	dc.DrawRoundedRectangle(rx, py, float64(W)-rx-s*24, s*86, s*12)
	dc.Stroke()

	sy := py + s*44
	base := rx + s*44
	drawStat(dc, s, base, sy, strconv.Itoa(d.Wins), "W", color.RGBA{34, 197, 94, 255})
	drawStat(dc, s, base+s*68, sy, strconv.Itoa(d.Draws), "D", color.RGBA{234, 179, 8, 255})
	drawStat(dc, s, base+s*136, sy, strconv.Itoa(d.Losses), "L", color.RGBA{239, 68, 68, 255})
	drawStat(dc, s, base+s*214, sy, fmt.Sprintf("%.0f%%", d.WinRate), "WIN", color.RGBA{129, 140, 248, 255})
	drawStat(dc, s, base+s*292, sy, strconv.Itoa(d.TotalGoals), "GOALS", color.RGBA{251, 146, 60, 255})

	// ── Нижняя акцентная полоса ──
	stripe := gg.NewLinearGradient(0, 0, float64(W), 0)
	stripe.AddColorStop(0, accent)
	stripe.AddColorStop(1, accent2)
	dc.SetFillStyle(stripe)
	dc.DrawRectangle(0, float64(H)-s*6, float64(W), s*6)
	dc.Fill()

	// ── Футер ──
	_ = loadFont(dc, fontBold, s*12)
	dc.SetRGBA(1, 1, 1, 0.3)
	dc.DrawStringAnchored("eFootLeague", float64(W)-s*20, float64(H)-s*22, 1, 0.5)

	var buf bytes.Buffer
	if err := encodePNGStdlib(&buf, dc.Image()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawStat(dc *gg.Context, s, x, y float64, val, label string, col color.Color) {
	_ = loadFont(dc, fontBold, s*20)
	dc.SetColor(col)
	dc.DrawStringAnchored(val, x, y, 0.5, 0.5)
	_ = loadFont(dc, fontRegular, s*10)
	dc.SetRGBA(1, 1, 1, 0.42)
	dc.DrawStringAnchored(label, x, y+s*18, 0.5, 0.5)
}

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

// accentColor — яркий брендовый цвет: цвет клуба, если он достаточно светлый,
// иначе второй цвет клуба, иначе акцент по рейтингу.
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

// secondColor — второй цвет для градиента (темнее/контрастный к accent).
func secondColor(club *clubdata.Club, accent color.RGBA) color.RGBA {
	if club != nil {
		c2 := hexColor(club.Color2)
		if colorDist(c2, accent) > 60 {
			return c2
		}
		c1 := hexColor(club.Color)
		if colorDist(c1, accent) > 60 {
			return c1
		}
	}
	return darken(accent, 0.55)
}

func colorDist(a, b color.RGBA) float64 {
	dr := float64(a.R) - float64(b.R)
	dg := float64(a.G) - float64(b.G)
	db := float64(a.B) - float64(b.B)
	return dr*dr + dg*dg + db*db // squared, ~3600 = «заметно разные»
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
	first := []rune(parts[0])
	out := string(first[0:1])
	if len(parts) > 1 {
		second := []rune(parts[1])
		out += string(second[0:1])
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
