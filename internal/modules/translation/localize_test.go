package translation

import (
	"strings"
	"testing"
)

func TestLocalize_CurrencyPToEN(t *testing.T) {
	text := "O produto custa R$ 1.234,56 hoje."
	out, res := Localize(text, "pt", "en")
	if !strings.Contains(out, "US$1,234.56") {
		t.Errorf("currency not converted: %s", out)
	}
	if res.Applied == 0 {
		t.Error("expected applied localization items")
	}
}

func TestLocalize_CurrencySimplePToEN(t *testing.T) {
	text := "Custa R$ 50 por mês."
	out, _ := Localize(text, "pt", "en")
	if !strings.Contains(out, "US$50.00") {
		t.Errorf("simple currency not converted: %s", out)
	}
}

func TestLocalize_CurrencyENToPT(t *testing.T) {
	text := "The product costs US$1,234.56 today."
	out, res := Localize(text, "en", "pt")
	if !strings.Contains(out, "R$1.234,56") {
		t.Errorf("currency not converted to PT: %s", out)
	}
	if res.Applied == 0 {
		t.Error("expected applied localization items")
	}
}

func TestLocalize_KmToMiles(t *testing.T) {
	text := "A cidade fica a 12 km daqui."
	out, _ := Localize(text, "pt", "en")
	if !strings.Contains(out, "mi") {
		t.Errorf("km not converted: %s", out)
	}
}

func TestLocalize_MilesToKm(t *testing.T) {
	text := "The city is 10 miles away."
	out, _ := Localize(text, "en", "pt")
	if !strings.Contains(out, "km") {
		t.Errorf("miles not converted: %s", out)
	}
}

func TestLocalize_CelsiusToFahrenheit(t *testing.T) {
	text := "A temperatura média é 25 °C."
	out, _ := Localize(text, "pt", "en")
	if !strings.Contains(out, "°F") {
		t.Errorf("celsius not converted: %s", out)
	}
}

func TestLocalize_DatePTtoEN(t *testing.T) {
	text := "O evento acontece em 05/08/2026."
	out, _ := Localize(text, "pt", "en")
	if !strings.Contains(out, "August 5, 2026") {
		t.Errorf("date not converted: %s", out)
	}
}

func TestLocalize_DateENtoPT(t *testing.T) {
	text := "The event happens on August 5, 2026."
	out, _ := Localize(text, "en", "pt")
	if !strings.Contains(out, "5 de agosto de 2026") {
		t.Errorf("date not converted to PT: %s", out)
	}
}

func TestLocalize_ExpressionNoBrasil(t *testing.T) {
	text := "Muitas empresas no Brasil usam essa técnica."
	out, res := Localize(text, "pt", "en")
	if !strings.Contains(out, "in the United States") {
		t.Errorf("expression not adapted: %s", out)
	}
	if res.Applied == 0 {
		t.Error("expected expression item")
	}
}

func TestLocalize_ExpressionUnitedStates(t *testing.T) {
	text := "Many companies in the United States use this technique."
	out, _ := Localize(text, "en", "pt")
	if !strings.Contains(out, "os Estados Unidos") {
		t.Errorf("expression not adapted to PT: %s", out)
	}
}

func TestLocalize_EnglishTextUnchanged(t *testing.T) {
	text := "This is a simple English sentence without any local markers."
	out, res := Localize(text, "pt", "en")
	if res.Applied != 0 {
		t.Errorf("expected no localizations, got %d", res.Applied)
	}
	if !strings.Contains(out, "This is a simple English sentence") {
		t.Errorf("text was modified unexpectedly: %s", out)
	}
}

func TestLocalize_Deterministic(t *testing.T) {
	text := "O total é R$ 100,00 e a distância é 5 km."
	o1, r1 := Localize(text, "pt", "en")
	o2, r2 := Localize(text, "pt", "en")
	if o1 != o2 || r1.Applied != r2.Applied {
		t.Error("localization not deterministic")
	}
}

func TestFormatUnit(t *testing.T) {
	if formatUnit(10, "mi") != "10 mi" {
		t.Errorf("whole unit formatting broken: %s", formatUnit(10, "mi"))
	}
	if formatUnit(7.456, "mi") != "7.5 mi" {
		t.Errorf("decimal unit formatting broken: %s", formatUnit(7.456, "mi"))
	}
}
