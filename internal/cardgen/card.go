package cardgen

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/color"

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
	Achievements []string // up to 5 icon strings
}

func GenerateCard(data CardData) ([]byte, error) {
	const W, H = 600, 350
	dc := gg.NewContext(W, H)

	bg1, bg2 := tierColors(data.Rating)
	drawGradient(dc, W, H, bg1, bg2)

	// Card border
	dc.SetRGBA(1, 1, 1, 0.08)
	dc.DrawRoundedRectangle(8, 8, W-16, H-16, 16)
	dc.SetLineWidth(1)
	dc.Stroke()

	// Avatar circle
	cx, cy := 90.0, 120.0
	r, g, b, _ := tierAccent(data.Rating).RGBA()
	dc.SetRGBA(float64(r)/65535, float64(g)/65535, float64(b)/65535, 0.9)
	dc.DrawCircle(cx, cy, 52)
	dc.Fill()

	// Initials
	_ = loadFont(dc, fontBold, 28)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(getInitials(data.DisplayName), cx, cy, 0.5, 0.4)

	// Club emoji below avatar
	if data.FavoriteClub != "" {
		_ = loadFont(dc, fontRegular, 20)
		dc.SetRGBA(1, 1, 1, 0.85)
		dc.DrawStringAnchored(data.FavoriteClub, cx, cy+66, 0.5, 0.5)
	}

	// Name
	_ = loadFont(dc, fontBold, 22)
	dc.SetRGBA(1, 1, 1, 0.95)
	dc.DrawString(truncate(data.DisplayName, 22), 165, 70)

	// Rank
	_ = loadFont(dc, fontRegular, 13)
	dc.SetRGBA(1, 1, 1, 0.55)
	dc.DrawString(data.Rank, 165, 95)

	// ELO value
	_ = loadFont(dc, fontBold, 36)
	dc.SetColor(tierAccent(data.Rating))
	dc.DrawString(fmt.Sprintf("%d", data.Rating), 165, 142)

	_ = loadFont(dc, fontRegular, 11)
	dc.SetRGBA(1, 1, 1, 0.35)
	dc.DrawString("ELO", 165, 158)

	// Stats row
	drawStat(dc, 165, 198, fmt.Sprintf("%d", data.Wins), "W", color.RGBA{34, 197, 94, 255})
	drawStat(dc, 228, 198, fmt.Sprintf("%d", data.Draws), "D", color.RGBA{234, 179, 8, 255})
	drawStat(dc, 291, 198, fmt.Sprintf("%d", data.Losses), "L", color.RGBA{239, 68, 68, 255})
	drawStat(dc, 374, 198, fmt.Sprintf("%.0f%%", data.WinRate), "WIN", color.RGBA{99, 102, 241, 255})
	drawStat(dc, 460, 198, fmt.Sprintf("%d", data.TotalGoals), "GOL", color.RGBA{251, 146, 60, 255})

	// Divider
	dc.SetRGBA(1, 1, 1, 0.08)
	dc.DrawLine(30, 242, W-30, 242)
	dc.SetLineWidth(1)
	dc.Stroke()

	// Achievement icons
	_ = loadFont(dc, fontRegular, 22)
	dc.SetRGBA(1, 1, 1, 0.75)
	ax := 52.0
	for i, icon := range data.Achievements {
		if i >= 5 {
			break
		}
		dc.DrawStringAnchored(icon, ax, 284, 0.5, 0.5)
		ax += 46
	}

	// Footer
	_ = loadFont(dc, fontBold, 10)
	dc.SetRGBA(1, 1, 1, 0.22)
	dc.DrawStringAnchored("eFootball League", float64(W)-18, float64(H)-14, 1, 0.5)

	var buf bytes.Buffer
	if err := encodePNGStdlib(&buf, dc.Image()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawStat(dc *gg.Context, x, y float64, val, label string, col color.Color) {
	_ = loadFont(dc, fontBold, 18)
	dc.SetColor(col)
	dc.DrawStringAnchored(val, x, y, 0.5, 0.5)
	_ = loadFont(dc, fontRegular, 9)
	dc.SetRGBA(1, 1, 1, 0.38)
	dc.DrawStringAnchored(label, x, y+16, 0.5, 0.5)
}

func drawGradient(dc *gg.Context, w, h int, c1, c2 color.RGBA) {
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		r := lerp(float64(c1.R), float64(c2.R), t)
		g := lerp(float64(c1.G), float64(c2.G), t)
		b := lerp(float64(c1.B), float64(c2.B), t)
		dc.SetRGB(r/255, g/255, b/255)
		dc.DrawLine(0, float64(y), float64(w), float64(y))
		dc.Stroke()
	}
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func tierColors(rating int) (color.RGBA, color.RGBA) {
	if rating >= 1500 {
		return color.RGBA{90, 60, 0, 255}, color.RGBA{10, 18, 35, 255}
	}
	if rating >= 1300 {
		return color.RGBA{20, 50, 100, 255}, color.RGBA{10, 18, 35, 255}
	}
	if rating >= 1150 {
		return color.RGBA{40, 20, 70, 255}, color.RGBA{10, 18, 35, 255}
	}
	return color.RGBA{18, 42, 18, 255}, color.RGBA{10, 18, 35, 255}
}

func tierAccent(rating int) color.Color {
	if rating >= 1500 {
		return color.RGBA{251, 191, 36, 255}
	}
	if rating >= 1300 {
		return color.RGBA{96, 165, 250, 255}
	}
	if rating >= 1150 {
		return color.RGBA{167, 139, 250, 255}
	}
	return color.RGBA{74, 222, 128, 255}
}

func getInitials(name string) string {
	runes := []rune(name)
	if len(runes) == 0 {
		return "?"
	}
	return string(runes[0:1])
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
