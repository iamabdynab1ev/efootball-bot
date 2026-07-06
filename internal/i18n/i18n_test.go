package i18n

import (
	"fmt"
	"strings"
	"testing"
)

func TestNotificationMessages(t *testing.T) {
	// Тексты уведомлений есть на всех трёх языках, аргументы подставляются.
	for _, lang := range []string{LangRu, LangUz, LangTg} {
		got := fmt.Sprintf(T(lang, "friendly.challenge.body"), "Али")
		if !strings.Contains(got, "Али") || strings.Contains(got, "%s") {
			t.Fatalf("lang=%s: аргумент не подставлен: %q", lang, got)
		}
	}
	// Неизвестный язык падает на узбекский (базовый язык пакета).
	if got, want := T("en", "friendly.confirmed.title"), T(LangUz, "friendly.confirmed.title"); got != want {
		t.Fatalf("fallback на uz не сработал: %q != %q", got, want)
	}
	// Неизвестный ключ возвращается как есть.
	if got := T(LangRu, "no.such.key"); got != "no.such.key" {
		t.Fatalf("неизвестный ключ: %q", got)
	}
	// У каждого текста уведомлений есть все три языка.
	for key, byLang := range notifMessages {
		for _, lang := range []string{LangRu, LangUz, LangTg} {
			if byLang[lang] == "" {
				t.Fatalf("ключ %s без перевода %s", key, lang)
			}
		}
	}
}
