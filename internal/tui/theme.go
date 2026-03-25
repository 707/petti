package tui

import "github.com/charmbracelet/lipgloss"

type ThemeName string

const (
	ThemeDefault      ThemeName = "default"
	ThemeEmber        ThemeName = "ember"
	ThemeFrost        ThemeName = "frost"
	ThemeDefaultDark  ThemeName = "default-dark"
	ThemeDefaultLight ThemeName = "default-light"
	ThemeEmberDark    ThemeName = "ember-dark"
	ThemeEmberLight   ThemeName = "ember-light"
	ThemeFrostDark    ThemeName = "frost-dark"
	ThemeFrostLight   ThemeName = "frost-light"
)

type themePalette struct {
	appForeground      lipgloss.Color
	mutedForeground    lipgloss.Color
	border             lipgloss.Color
	panelBackground    lipgloss.Color
	headerBackground   lipgloss.Color
	headerForeground   lipgloss.Color
	topBackground      lipgloss.Color
	topForeground      lipgloss.Color
	selectedBackground lipgloss.Color
	selectedForeground lipgloss.Color
	statusForeground   lipgloss.Color
	homebrewForeground lipgloss.Color
	caskForeground     lipgloss.Color
	npmForeground      lipgloss.Color
	pipForeground      lipgloss.Color
	dependencyYes      lipgloss.Color
	dependencyNo       lipgloss.Color
}

func validThemes() []ThemeName {
	return []ThemeName{
		ThemeDefault, ThemeEmber, ThemeFrost,
		ThemeDefaultDark, ThemeDefaultLight,
		ThemeEmberDark, ThemeEmberLight,
		ThemeFrostDark, ThemeFrostLight,
	}
}

func isValidTheme(theme ThemeName) bool {
	for _, candidate := range validThemes() {
		if theme == candidate {
			return true
		}
	}
	return false
}

func IsValidTheme(theme ThemeName) bool {
	return isValidTheme(theme)
}

func ValidThemes() []ThemeName {
	return append([]ThemeName(nil), validThemes()...)
}

func nextTheme(theme ThemeName) ThemeName {
	themes := []ThemeName{
		ThemeDefaultDark,
		ThemeDefaultLight,
		ThemeEmberDark,
		ThemeEmberLight,
		ThemeFrostDark,
		ThemeFrostLight,
	}
	theme = normalizeTheme(theme)
	for index, candidate := range themes {
		if candidate != theme {
			continue
		}
		return themes[(index+1)%len(themes)]
	}
	return ThemeDefaultDark
}

func paletteForTheme(theme ThemeName) themePalette {
	switch normalizeTheme(theme) {
	case ThemeEmberDark:
		return themePalette{
			appForeground:      lipgloss.Color("252"),
			mutedForeground:    lipgloss.Color("244"),
			border:             lipgloss.Color("209"),
			panelBackground:    lipgloss.Color("235"),
			headerBackground:   lipgloss.Color("88"),
			headerForeground:   lipgloss.Color("230"),
			topBackground:      lipgloss.Color("52"),
			topForeground:      lipgloss.Color("230"),
			selectedBackground: lipgloss.Color("166"),
			selectedForeground: lipgloss.Color("255"),
			statusForeground:   lipgloss.Color("223"),
			homebrewForeground: lipgloss.Color("216"),
			caskForeground:     lipgloss.Color("209"),
			npmForeground:      lipgloss.Color("180"),
			pipForeground:      lipgloss.Color("223"),
			dependencyYes:      lipgloss.Color("221"),
			dependencyNo:       lipgloss.Color("114"),
		}
	case ThemeEmberLight:
		return themePalette{
			appForeground:      lipgloss.Color("235"),
			mutedForeground:    lipgloss.Color("240"),
			border:             lipgloss.Color("173"),
			panelBackground:    lipgloss.Color("255"),
			headerBackground:   lipgloss.Color("216"),
			headerForeground:   lipgloss.Color("235"),
			topBackground:      lipgloss.Color("223"),
			topForeground:      lipgloss.Color("235"),
			selectedBackground: lipgloss.Color("215"),
			selectedForeground: lipgloss.Color("235"),
			statusForeground:   lipgloss.Color("130"),
			homebrewForeground: lipgloss.Color("130"),
			caskForeground:     lipgloss.Color("166"),
			npmForeground:      lipgloss.Color("137"),
			pipForeground:      lipgloss.Color("94"),
			dependencyYes:      lipgloss.Color("172"),
			dependencyNo:       lipgloss.Color("64"),
		}
	case ThemeFrostDark:
		return themePalette{
			appForeground:      lipgloss.Color("255"),
			mutedForeground:    lipgloss.Color("250"),
			border:             lipgloss.Color("117"),
			panelBackground:    lipgloss.Color("236"),
			headerBackground:   lipgloss.Color("31"),
			headerForeground:   lipgloss.Color("255"),
			topBackground:      lipgloss.Color("24"),
			topForeground:      lipgloss.Color("255"),
			selectedBackground: lipgloss.Color("39"),
			selectedForeground: lipgloss.Color("255"),
			statusForeground:   lipgloss.Color("153"),
			homebrewForeground: lipgloss.Color("117"),
			caskForeground:     lipgloss.Color("111"),
			npmForeground:      lipgloss.Color("153"),
			pipForeground:      lipgloss.Color("87"),
			dependencyYes:      lipgloss.Color("186"),
			dependencyNo:       lipgloss.Color("120"),
		}
	case ThemeFrostLight:
		return themePalette{
			appForeground:      lipgloss.Color("236"),
			mutedForeground:    lipgloss.Color("243"),
			border:             lipgloss.Color("110"),
			panelBackground:    lipgloss.Color("255"),
			headerBackground:   lipgloss.Color("153"),
			headerForeground:   lipgloss.Color("236"),
			topBackground:      lipgloss.Color("189"),
			topForeground:      lipgloss.Color("236"),
			selectedBackground: lipgloss.Color("117"),
			selectedForeground: lipgloss.Color("255"),
			statusForeground:   lipgloss.Color("67"),
			homebrewForeground: lipgloss.Color("31"),
			caskForeground:     lipgloss.Color("38"),
			npmForeground:      lipgloss.Color("67"),
			pipForeground:      lipgloss.Color("24"),
			dependencyYes:      lipgloss.Color("172"),
			dependencyNo:       lipgloss.Color("35"),
		}
	case ThemeDefaultLight:
		return themePalette{
			appForeground:      lipgloss.Color("236"),
			mutedForeground:    lipgloss.Color("243"),
			border:             lipgloss.Color("147"),
			panelBackground:    lipgloss.Color("255"),
			headerBackground:   lipgloss.Color("183"),
			headerForeground:   lipgloss.Color("236"),
			topBackground:      lipgloss.Color("189"),
			topForeground:      lipgloss.Color("236"),
			selectedBackground: lipgloss.Color("141"),
			selectedForeground: lipgloss.Color("255"),
			statusForeground:   lipgloss.Color("98"),
			homebrewForeground: lipgloss.Color("99"),
			caskForeground:     lipgloss.Color("134"),
			npmForeground:      lipgloss.Color("104"),
			pipForeground:      lipgloss.Color("61"),
			dependencyYes:      lipgloss.Color("178"),
			dependencyNo:       lipgloss.Color("65"),
		}
	default:
		return themePalette{
			appForeground:      lipgloss.Color("252"),
			mutedForeground:    lipgloss.Color("248"),
			border:             lipgloss.Color("141"),
			panelBackground:    lipgloss.Color("236"),
			headerBackground:   lipgloss.Color("57"),
			headerForeground:   lipgloss.Color("255"),
			topBackground:      lipgloss.Color("238"),
			topForeground:      lipgloss.Color("255"),
			selectedBackground: lipgloss.Color("63"),
			selectedForeground: lipgloss.Color("255"),
			statusForeground:   lipgloss.Color("220"),
			homebrewForeground: lipgloss.Color("147"),
			caskForeground:     lipgloss.Color("183"),
			npmForeground:      lipgloss.Color("153"),
			pipForeground:      lipgloss.Color("110"),
			dependencyYes:      lipgloss.Color("221"),
			dependencyNo:       lipgloss.Color("114"),
		}
	}
}

func normalizeTheme(theme ThemeName) ThemeName {
	switch theme {
	case ThemeDefault, "":
		return ThemeDefaultDark
	case ThemeEmber:
		return ThemeEmberDark
	case ThemeFrost:
		return ThemeFrostDark
	default:
		return theme
	}
}
