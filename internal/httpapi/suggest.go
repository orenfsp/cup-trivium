package httpapi

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"otklik/internal/store"
)

// categorySuggestion — подсказка системы по категории обращения (ТЗ п.4.2):
// оператору в окне обработки показывается, куда система относит обращение
// и по каким маркерам. Подсказка не заменяет решение оператора.
type categorySuggestion struct {
	CategoryID      uuid.UUID `json:"category_id"`
	CategoryName    string    `json:"category_name"`
	SpecialistGroup string    `json:"specialist_group"`
	MatchedKeywords []string  `json:"matched_keywords"`
}

// groupMarkers — маркеры-подстроки (в нижнем регистре, без окончаний),
// по которым текст обращения относится к профильной группе специалистов.
var groupMarkers = map[string][]string{
	"psychologists": {
		"травл", "буллинг", "дразн", "оскорбл", "обзыв", "угроз", "гроз", "страш",
		"тревог", "паник", "плач", "депресс", "апати", "одинок", "самооценк",
		"боюсь", "боящ", "обид", "давлен", "выгор", "устал", "не хочу", "ноч",
		"не сплю", "спать не", "злост", "мстит", "униж", "насмеха", "исключ",
	},
	"conflictologists": {
		"конфликт", "ссор", "поругал", "руга", "спор", "драк", "не общаем",
		"не говорит", "обвиня", "миров", "договор", "примир", "раздор",
	},
	"lawyers": {
		"юрист", "прав", "закон", "полиц", "суд", "иск", "заявлен", "штраф",
		"вымогател", "краж", "украл", "ответственност", "компенсац", "доказат",
		"видеозапис", "порча",
	},
	"social_pedagogues": {
		"трудн", "жизненн", "деньг", "финанс", "жиль", "пособи", "опек",
		"переезд", "перевел", "болезн", "инвалид", "многодет", "потерял",
	},
}

// groupTitles — русские названия профильных групп для подсказки оператору.
var groupTitles = map[string]string{
	"psychologists":       "психолог",
	"conflictologists":    "конфликтолог",
	"lawyers":             "юрист",
	"social_pedagogues":   "социальный педагог",
}

// suggestCategory скорит профильные группы по маркерам в тексте (описание +
// ответы анкеты), затем внутри лучшей группы выбирает категорию по
// совпадению слов её названия. Возвращает nil, если маркеров не нашлось.
func (s *Server) suggestCategory(ctx context.Context, a store.Appeal, answers []store.IntakeAnswer) *categorySuggestion {
	var b strings.Builder
	b.WriteString(a.Description)
	for _, ans := range answers {
		b.WriteString(" " + ans.Answer)
	}
	text := strings.ToLower(b.String())

	bestGroup, bestScore := "", 0
	var matched []string
	for group, markers := range groupMarkers {
		score := 0
		var hits []string
		for _, m := range markers {
			if strings.Contains(text, m) {
				score++
				hits = append(hits, strings.TrimSpace(m))
			}
		}
		if score > bestScore {
			bestGroup, bestScore, matched = group, score, hits
		}
	}
	if bestGroup == "" {
		return nil
	}

	cats, err := s.st.ListCategoriesPublic(ctx)
	if err != nil {
		return nil
	}
	// Категория в группе: та, чьё название сильнее всего пересекается с текстом.
	var bestCat *store.Category
	bestCatScore := -1
	for i := range cats {
		c := cats[i]
		if c.SpecialistGroup != bestGroup {
			continue
		}
		score := 0
		for _, w := range strings.Fields(strings.ToLower(c.Name)) {
			if len([]rune(w)) >= 5 && strings.Contains(text, w) {
				score++
			}
		}
		if score > bestCatScore {
			bestCat, bestCatScore = &cats[i], score
		}
	}
	if bestCat == nil {
		return nil
	}
	return &categorySuggestion{
		CategoryID:      bestCat.ID,
		CategoryName:    bestCat.Name,
		SpecialistGroup: groupTitles[bestGroup],
		MatchedKeywords: matched,
	}
}
