package strataudit

import (
	"fmt"
	"testing"
)

func TestFindingTemplates_AllTypes(t *testing.T) {
	types := []string{"gap", "orphan", "strong_trace", "alignment", "weak_link", "ambiguous", "coverage"}
	for _, ft := range types {
		t.Run(ft, func(t *testing.T) {
			ru := tpl("ru", ft)
			en := tpl("en", ft)
			if ru.Title == "" {
				t.Error("ru title empty")
			}
			if en.Title == "" {
				t.Error("en title empty")
			}
		})
	}
}

func TestFindingTemplates_RussianGap(t *testing.T) {
	tpl := tpl("ru", "gap")
	title := fmt.Sprintf(tpl.Title, "Цель XYZ", "initiative")
	if title == "" {
		t.Fatal("empty title")
	}
	// Should contain Cyrillic
	found := false
	for _, r := range title {
		if r >= 0x0400 && r <= 0x04FF {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("title has no Cyrillic: %q", title)
	}
}

func TestFindingTemplates_Fallback(t *testing.T) {
	// Unknown language should fall back to English
	got := tpl("xx", "gap")
	if got.Title == "" {
		t.Error("expected English fallback")
	}
	en := tpl("en", "gap")
	if got.Title != en.Title {
		t.Errorf("got %q, want English fallback %q", got.Title, en.Title)
	}
}

func TestConfigDefaultLang(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()
	if cfg.Output.Lang != "ru" {
		t.Errorf("default lang = %q, want 'ru'", cfg.Output.Lang)
	}
}
