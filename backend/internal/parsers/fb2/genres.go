package fb2

import (
	"strings"

	"boyan/internal/models"
)

// genreMap сопоставляет коды FictionBook с человекочитаемыми категориями на русском и английском.
var genreMap = map[string]models.Genre{
	// Фантастика / Science Fiction & Fantasy
	"sf":                 {Code: "sf", NameRU: "Научная фантастика", NameEN: "Science Fiction", CategoryRU: "Фантастика", CategoryEN: "Sci-Fi & Fantasy"},
	"sf_history":         {Code: "sf_history", NameRU: "Альтернативная история", NameEN: "Alternative History", CategoryRU: "Фантастика", CategoryEN: "Sci-Fi & Fantasy"},
	"sf_action":          {Code: "sf_action", NameRU: "Боевая фантастика", NameEN: "Action Sci-Fi", CategoryRU: "Фантастика", CategoryEN: "Sci-Fi & Fantasy"},
	"sf_heroic":          {Code: "sf_heroic", NameRU: "Героическая фантастика", NameEN: "Heroic Fantasy", CategoryRU: "Фантастика", CategoryEN: "Sci-Fi & Fantasy"},
	"sf_fantasy":         {Code: "sf_fantasy", NameRU: "Фэнтези", NameEN: "Fantasy", CategoryRU: "Фантастика", CategoryEN: "Sci-Fi & Fantasy"},
	"sf_space":           {Code: "sf_space", NameRU: "Космическая фантастика", NameEN: "Space Opera", CategoryRU: "Фантастика", CategoryEN: "Sci-Fi & Fantasy"},
	"sf_cyberpunk":       {Code: "sf_cyberpunk", NameRU: "Киберпанк", NameEN: "Cyberpunk", CategoryRU: "Фантастика", CategoryEN: "Sci-Fi & Fantasy"},
	"sf_postapocalyptic": {Code: "sf_postapocalyptic", NameRU: "Постапокалипсис", NameEN: "Post-Apocalyptic", CategoryRU: "Фантастика", CategoryEN: "Sci-Fi & Fantasy"},
	"popadanec":          {Code: "popadanec", NameRU: "Попаданцы", NameEN: "Time/World Travel", CategoryRU: "Фантастика", CategoryEN: "Sci-Fi & Fantasy"},

	// Детективы и триллеры
	"det_action":    {Code: "det_action", NameRU: "Боевик", NameEN: "Action Detective", CategoryRU: "Детективы и Триллеры", CategoryEN: "Detectives & Thrillers"},
	"det_ironic":    {Code: "det_ironic", NameRU: "Иронический детектив", NameEN: "Cozy Mystery", CategoryRU: "Детективы и Триллеры", CategoryEN: "Detectives & Thrillers"},
	"det_history":   {Code: "det_history", NameRU: "Исторический детектив", NameEN: "Historical Mystery", CategoryRU: "Детективы и Триллеры", CategoryEN: "Detectives & Thrillers"},
	"det_classic":   {Code: "det_classic", NameRU: "Классический детектив", NameEN: "Classic Mystery", CategoryRU: "Детективы и Триллеры", CategoryEN: "Detectives & Thrillers"},
	"det_crime":     {Code: "det_crime", NameRU: "Криминальный детектив", NameEN: "Crime", CategoryRU: "Детективы и Триллеры", CategoryEN: "Detectives & Thrillers"},
	"det_police":    {Code: "det_police", NameRU: "Полицейский детектив", NameEN: "Police Procedural", CategoryRU: "Детективы и Триллеры", CategoryEN: "Detectives & Thrillers"},
	"thriller":      {Code: "thriller", NameRU: "Триллер", NameEN: "Thriller", CategoryRU: "Детективы и Триллеры", CategoryEN: "Detectives & Thrillers"},

	// Проза
	"prose_classic":      {Code: "prose_classic", NameRU: "Классическая проза", NameEN: "Classic Prose", CategoryRU: "Проза", CategoryEN: "Prose"},
	"prose_history":      {Code: "prose_history", NameRU: "Историческая проза", NameEN: "Historical Fiction", CategoryRU: "Проза", CategoryEN: "Prose"},
	"prose_contemporary": {Code: "prose_contemporary", NameRU: "Современная проза", NameEN: "Contemporary Fiction", CategoryRU: "Проза", CategoryEN: "Prose"},
	"prose_military":     {Code: "prose_military", NameRU: "Военная проза", NameEN: "War Fiction", CategoryRU: "Проза", CategoryEN: "Prose"},

	// Любовные романы
	"love_contemporary": {Code: "love_contemporary", NameRU: "Современные любовные романы", NameEN: "Contemporary Romance", CategoryRU: "Любовные романы", CategoryEN: "Romance"},
	"love_history":      {Code: "love_history", NameRU: "Исторические любовные романы", NameEN: "Historical Romance", CategoryRU: "Любовные романы", CategoryEN: "Romance"},
	"love_fantasy":      {Code: "love_fantasy", NameRU: "Любовно-фантастические романы", NameEN: "Fantasy Romance", CategoryRU: "Любовные романы", CategoryEN: "Romance"},

	// Деловая литература и наука
	"sci_history":   {Code: "sci_history", NameRU: "История", NameEN: "History", CategoryRU: "Наука и Образование", CategoryEN: "Science & Education"},
	"sci_philosophy":{Code: "sci_philosophy", NameRU: "Философия", NameEN: "Philosophy", CategoryRU: "Наука и Образование", CategoryEN: "Science & Education"},
	"sci_psychology":{Code: "sci_psychology", NameRU: "Психология", NameEN: "Psychology", CategoryRU: "Наука и Образование", CategoryEN: "Science & Education"},
	"sci_business":  {Code: "sci_business", NameRU: "Деловая литература", NameEN: "Business", CategoryRU: "Бизнес", CategoryEN: "Business"},
	"computers":     {Code: "computers", NameRU: "Околокомпьютерная литература", NameEN: "Computers & Tech", CategoryRU: "Техника и IT", CategoryEN: "Technology & IT"},
}

// NormalizeGenre преобразует код жанра FB2 в нормализованную структуру с переводом.
func NormalizeGenre(rawCode string) models.Genre {
	code := strings.ToLower(strings.TrimSpace(rawCode))
	if g, ok := genreMap[code]; ok {
		return g
	}

	// Если код нестандартный, формируем читаемое имя
	readable := strings.ReplaceAll(code, "_", " ")
	readable = strings.Title(readable)

	return models.Genre{
		Code:        code,
		NameRU:      readable,
		NameEN:      readable,
		CategoryRU:  "Прочее",
		CategoryEN:  "Other",
	}
}
