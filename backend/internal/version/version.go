package version

// Version представляет текущую семантическую версию приложения.
const Version = "0.0.1"

// Stage определяет стадию выпуска (например, "alpha", "beta", "rc", "").
const Stage = "beta"

// Full возвращает полную строку версии с суффиксом стадии выпуска.
func Full() string {
	if Stage != "" {
		return Version + "-" + Stage
	}
	return Version
}
