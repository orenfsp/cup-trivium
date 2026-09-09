package domain

import "strings"

// CrisisMarkers — словарь кризисных маркеров (P0-эвристика, не ML).
var CrisisMarkers = []string{
	"убить", "убью", "убива", "суицид", "самоубийств", "не хочу жить",
	"покончить с собой", "покончила с собой", "всё надоело", "устал жить",
	"хочу умереть", "не хочу жить", "исчезнуть навсегда",
	"угрожают", "избивают", "избил", "физическое насилие",
	"насилие", "оружие", "нож", "душит", "бьет", "бьёт", "умру",
}

func normalizeCrisisText(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "ё", "е")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func DetectCrisis(texts ...string) bool {
	for _, raw := range texts {
		t := normalizeCrisisText(raw)
		if t == "" {
			continue
		}
		for _, m := range CrisisMarkers {
			if m != "" && strings.Contains(t, normalizeCrisisText(m)) {
				return true
			}
		}
	}
	return false
}

type CrisisHelp struct {
	Title       string `json:"title"`
	Phone       string `json:"phone"`
	Description string `json:"description"`
}

var CrisisHelpContacts = []CrisisHelp{
	{
		Title:       "Детский телефон доверия (бесплатно, круглосуточно, анонимно)",
		Phone:       "8-800-2000-122",
		Description: "Единый детский телефон доверия: психологическая помощь детям, подросткам и родителям.",
	},
	{
		Title:       "Единый номер экстренных служб",
		Phone:       "112",
		Description: "При угрозе жизни и здоровью звоните немедленно: единый диспетчер экстренных служб.",
	},
	{
		Title:       "Полиция",
		Phone:       "102",
		Description: "Вызов полиции при преступлении, насилии или угрозе (с мобильного и городского телефона).",
	},
	{
		Title:       "Скорая медицинская помощь",
		Phone:       "103",
		Description: "Вызов скорой помощи при травмах и прямой угрозе здоровью.",
	},
	{
		Title:       "Телефон доверия ГУ МВД России по Оренбургской области",
		Phone:       "8 (3532) 79-00-22",
		Description: "Сообщить о преступлении против несовершеннолетнего (Оренбургская область).",
	},
}
