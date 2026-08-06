package translation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Deterministic cultural localization: currency, units of measure, date formats
// and safe country/expression adaptations, applied to the TRANSLATED text so the
// target locale feels native. All rules are pure functions of the input text.

type LocalizationResult struct {
	Applied int
	Items   []string
}

var (
	brCurrencyRe = regexp.MustCompile(`R\$\s*(\d{1,3}(?:\.\d{3})*(?:,\d+)?|\d+(?:,\d+)?)`)
	usCurrencyRe = regexp.MustCompile(`US\$\s*(\d{1,3}(?:,\d{3})*(?:\.\d+)?|\d+(?:\.\d+)?)`)

	kmRe      = regexp.MustCompile(`(?i)(\d{1,3}(?:\.\d{3})*(?:,\d+)?|\d+(?:,\d+)?)\s*(?:km|quilômetros?|quilometros?)`)
	miRe      = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(?:mi|miles?)\b`)
	mRe       = regexp.MustCompile(`(?i)(\d{1,3}(?:\.\d{3})*(?:,\d+)?|\d+(?:,\d+)?)\s*m\b`)
	ftRe      = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(?:ft|feet)\b`)
	kgRe      = regexp.MustCompile(`(?i)(\d{1,3}(?:\.\d{3})*(?:,\d+)?|\d+(?:,\d+)?)\s*(?:kg|quilogramas?)`)
	lbRe      = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(?:lb|lbs|pounds?)\b`)
	celsiusRe = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*°\s*C`)
	fahrRe    = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*°\s*F`)

	ptDateRe = regexp.MustCompile(`\b(\d{1,2})/(\d{1,2})/(\d{4})\b`)
	enDateRe = regexp.MustCompile(`\b(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2})(?:st|nd|rd|th)?\s*,?\s+(\d{4})\b`)
)

var ptMonthNames = []string{"janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}

var enMonthNames = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}

func parseBRNumber(s string) (float64, bool) {
	clean := strings.ReplaceAll(strings.ReplaceAll(s, ".", ""), ",", ".")
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func parseUSNumber(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func formatWithThousands(v float64, decimals int) string {
	// deterministic formatting with thousands separators
	intPart := int64(v)
	frac := v - float64(intPart)
	decimalsStr := ""
	if decimals > 0 {
		mult := 1.0
		for i := 0; i < decimals; i++ {
			mult *= 10
		}
		decimalsStr = fmt.Sprintf(".%0*d", decimals, int64(frac*mult+0.5))
	}
	// group integer part in thousands
	s := strconv.FormatInt(intPart, 10)
	var groups []string
	for len(s) > 3 {
		groups = append([]string{s[len(s)-3:]}, groups...)
		s = s[:len(s)-3]
	}
	groups = append([]string{s}, groups...)
	return strings.Join(groups, ",") + decimalsStr
}

// Localize applies locale-specific conversions to already-translated text.
func Localize(text, fromLang, toLang string) (string, LocalizationResult) {
	var res LocalizationResult

	if toLang == "en" {
		text = brCurrencyRe.ReplaceAllStringFunc(text, func(m string) string {
			num := brCurrencyRe.FindStringSubmatch(m)
			if num == nil {
				return m
			}
			v, ok := parseBRNumber(num[1])
			if !ok {
				return m
			}
			item := "currency: R$" + num[1] + " -> US$" + formatWithThousands(v, 2)
			res.Items = append(res.Items, item)
			return "US$" + formatWithThousands(v, 2)
		})

		text = kmRe.ReplaceAllStringFunc(text, func(m string) string {
			num := kmRe.FindStringSubmatch(m)
			if num == nil {
				return m
			}
			v, ok := parseBRNumber(num[1])
			if !ok {
				return m
			}
			miles := v * 0.621371
			out := formatUnit(miles, "mi")
			res.Items = append(res.Items, fmt.Sprintf("unit: %s km -> %s", num[1], out))
			return out
		})

		text = mRe.ReplaceAllStringFunc(text, func(m string) string {
			num := mRe.FindStringSubmatch(m)
			if num == nil {
				return m
			}
			v, ok := parseBRNumber(num[1])
			if !ok {
				return m
			}
			out := formatUnit(v*3.28084, "ft")
			res.Items = append(res.Items, fmt.Sprintf("unit: %s m -> %s", num[1], out))
			return out
		})

		text = kgRe.ReplaceAllStringFunc(text, func(m string) string {
			num := kgRe.FindStringSubmatch(m)
			if num == nil {
				return m
			}
			v, ok := parseBRNumber(num[1])
			if !ok {
				return m
			}
			out := formatUnit(v*2.20462, "lb")
			res.Items = append(res.Items, fmt.Sprintf("unit: %s kg -> %s", num[1], out))
			return out
		})

		text = celsiusRe.ReplaceAllStringFunc(text, func(m string) string {
			num := celsiusRe.FindStringSubmatch(m)
			if num == nil {
				return m
			}
			v, err := strconv.ParseFloat(num[1], 64)
			if err != nil {
				return m
			}
			out := formatUnit(v*9/5+32, "°F")
			res.Items = append(res.Items, fmt.Sprintf("unit: %s °C -> %s", num[1], out))
			return out
		})

		text = ptDateRe.ReplaceAllStringFunc(text, func(m string) string {
			num := ptDateRe.FindStringSubmatch(m)
			if num == nil {
				return m
			}
			day, _ := strconv.Atoi(num[1])
			month, _ := strconv.Atoi(num[2])
			if month < 1 || month > 12 {
				return m
			}
			out := fmt.Sprintf("%s %d, %s", enMonthNames[month-1], day, num[3])
			res.Items = append(res.Items, fmt.Sprintf("date: %s -> %s", m, out))
			return out
		})

		// Safe phrase adaptations
		adaptations := map[string]string{
			"(?i)\\bno brasil\\b":    "in the United States",
			"(?i)\\bdo brasil\\b":    "of the United States",
			"(?i)\\bbrasileiro\\b":   "American",
			"(?i)\\bbrasileiros\\b":  "Americans",
			"(?i)\\bbrasileira\\b":   "American",
			"(?i)\\bbrasileiras\\b":  "Americans",
		}
		for pat, repl := range adaptations {
			re := regexp.MustCompile(pat)
			if re.MatchString(text) {
				text = re.ReplaceAllString(text, repl)
				res.Items = append(res.Items, fmt.Sprintf("expression: %s -> %s", pat, repl))
			}
		}
	}

	if toLang == "pt" {
		text = usCurrencyRe.ReplaceAllStringFunc(text, func(m string) string {
			num := usCurrencyRe.FindStringSubmatch(m)
			if num == nil {
				return m
			}
			v, ok := parseUSNumber(num[1])
			if !ok {
				return m
			}
			// Brazilian format: dots for thousands, comma for decimals
			s := strconv.FormatFloat(v, 'f', 2, 64)
			sign := ""
			if strings.HasPrefix(s, "-") {
				sign = "-"
				s = s[1:]
			}
			parts := strings.Split(s, ".")
			intPart := parts[0]
			var groups []string
			for len(intPart) > 3 {
				groups = append([]string{intPart[len(intPart)-3:]}, groups...)
				intPart = intPart[:len(intPart)-3]
			}
			groups = append([]string{intPart}, groups...)
			out := "R$" + sign + strings.Join(groups, ".") + "," + parts[1]
			res.Items = append(res.Items, fmt.Sprintf("currency: %s -> %s", m, out))
			return out
		})

		text = miRe.ReplaceAllStringFunc(text, func(m string) string {
			num := miRe.FindStringSubmatch(m)
			if num == nil {
				return m
			}
			v, err := strconv.ParseFloat(num[1], 64)
			if err != nil {
				return m
			}
			out := formatUnit(v*1.609344, "km")
			res.Items = append(res.Items, fmt.Sprintf("unit: %s mi -> %s", num[1], out))
			return out
		})

		text = ftRe.ReplaceAllStringFunc(text, func(m string) string {
			num := ftRe.FindStringSubmatch(m)
			if num == nil {
				return m
			}
			v, err := strconv.ParseFloat(num[1], 64)
			if err != nil {
				return m
			}
			out := formatUnit(v*0.3048, "m")
			res.Items = append(res.Items, fmt.Sprintf("unit: %s ft -> %s", num[1], out))
			return out
		})

		text = lbRe.ReplaceAllStringFunc(text, func(m string) string {
			num := lbRe.FindStringSubmatch(m)
			if num == nil {
				return m
			}
			v, err := strconv.ParseFloat(num[1], 64)
			if err != nil {
				return m
			}
			out := formatUnit(v*0.453592, "kg")
			res.Items = append(res.Items, fmt.Sprintf("unit: %s lb -> %s", num[1], out))
			return out
		})

		text = fahrRe.ReplaceAllStringFunc(text, func(m string) string {
			num := fahrRe.FindStringSubmatch(m)
			if num == nil {
				return m
			}
			v, err := strconv.ParseFloat(num[1], 64)
			if err != nil {
				return m
			}
			out := formatUnit((v-32)*5/9, "°C")
			res.Items = append(res.Items, fmt.Sprintf("unit: %s °F -> %s", num[1], out))
			return out
		})

		text = enDateRe.ReplaceAllStringFunc(text, func(m string) string {
			num := enDateRe.FindStringSubmatch(m)
			if num == nil {
				return m
			}
			monthIdx := -1
			for i, name := range enMonthNames {
				if strings.EqualFold(name, num[1]) {
					monthIdx = i
					break
				}
			}
			if monthIdx < 0 {
				return m
			}
			out := fmt.Sprintf("%s de %s de %s", num[2], ptMonthNames[monthIdx], num[3])
			res.Items = append(res.Items, fmt.Sprintf("date: %s -> %s", m, out))
			return out
		})

		adaptations := map[string]string{
			"(?i)\\bthe united states\\b": "os Estados Unidos",
			"(?i)\\bamericans\\b":         "americanos",
			"(?i)\\bamerican\\b":          "americano",
		}
		for pat, repl := range adaptations {
			re := regexp.MustCompile(pat)
			if re.MatchString(text) {
				text = re.ReplaceAllString(text, repl)
				res.Items = append(res.Items, fmt.Sprintf("expression: %s -> %s", pat, repl))
			}
		}
	}

	res.Applied = len(res.Items)
	return text, res
}

// formatUnit renders a converted measurement with a sensible number of decimals
// (integer when whole, otherwise one decimal), deterministic.
func formatUnit(v float64, unit string) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d %s", int64(v), unit)
	}
	return fmt.Sprintf("%.1f %s", v, unit)
}
