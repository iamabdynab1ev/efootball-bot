#!/usr/bin/env bash
# Генерация всех иконок приложения из одного исходника логотипа.
# Использование: ./scripts/make-icons.sh path/to/logo.png
# Требует ImageMagick (magick).
set -euo pipefail

SRC="${1:?Укажите путь к исходному логотипу (квадратный PNG/JPG)}"
OUT="$(cd "$(dirname "$0")/.." && pwd)/web/public"

# Квадрат + основные размеры PWA/иконок.
magick "$SRC" -resize 1024x1024^ -gravity center -extent 1024x1024 /tmp/logo-sq.png

magick /tmp/logo-sq.png -resize 512x512 "$OUT/icon-512.png"
magick /tmp/logo-sq.png -resize 192x192 "$OUT/icon-192.png"
# apple-touch: iOS сам скругляет углы; фон нужен непрозрачный.
magick /tmp/logo-sq.png -resize 180x180 -background black -alpha remove "$OUT/apple-touch-icon.png"
# badge для push-статус-бара: монохромный белый силуэт на прозрачном.
magick /tmp/logo-sq.png -resize 96x96 -colorspace Gray -level 25%,75% \
  -alpha off \( +clone -fx 'u' \) -compose CopyOpacity -composite \
  -fill white -colorize 100 "$OUT/badge.png"

echo "Готово: icon-512, icon-192, apple-touch-icon, badge → $OUT"
