package mail

import "strings"

// Texts of Wokko's mails, plain and short. Each language has every kind; others fall back to
// English. {link}, {name}, {site} and {hours} are filled in.
var texts = map[string]map[string][2]string{
	"en": {
		"invite":  {"You're invited to {site}", "Hello,\n\nyou're invited to {site}, where we keep our recipes.\n\nCreate your account here:\n{link}\n\nIf you didn't expect this, you can ignore this mail."},
		"setup":   {"Your account on {site}", "Hello {name},\n\nan account on {site} is ready for you. Choose your password here within 7 days:\n{link}\n\nIf you didn't expect this, you can ignore this mail."},
		"reset":   {"Choose a new password for {site}", "Hello {name},\n\nchoose a new password for {site} here:\n{link}\n\nThe link works once and for {hours} hours. If you didn't ask for it, you can ignore this mail; your password stays as it is."},
		"confirm": {"Confirm your email address for {site}", "Hello {name},\n\nplease confirm that this is your email address for {site}:\n{link}\n\nIf you didn't add it, you can ignore this mail."},
		"test":    {"Test mail from {site}", "Hello {name},\n\nthis is a test mail from {site}. Mail works."},
	},
	"de": {
		"invite":  {"Einladung zu {site}", "Hallo,\n\ndu bist zu {site} eingeladen, wo wir unsere Rezepte sammeln.\n\nHier legst du dein Konto an:\n{link}\n\nWenn du damit nicht gerechnet hast, kannst du diese Mail ignorieren."},
		"setup":   {"Dein Konto bei {site}", "Hallo {name},\n\nbei {site} ist ein Konto für dich bereit. Wähle hier innerhalb von 7 Tagen dein Passwort:\n{link}\n\nWenn du damit nicht gerechnet hast, kannst du diese Mail ignorieren."},
		"reset":   {"Neues Passwort für {site}", "Hallo {name},\n\nhier wählst du ein neues Passwort für {site}:\n{link}\n\nDer Link funktioniert einmal und {hours} Stunden lang. Wenn du ihn nicht angefordert hast, kannst du diese Mail ignorieren; dein Passwort bleibt, wie es ist."},
		"confirm": {"Bestätige deine E-Mail-Adresse für {site}", "Hallo {name},\n\nbitte bestätige, dass das deine E-Mail-Adresse für {site} ist:\n{link}\n\nWenn du sie nicht eingetragen hast, kannst du diese Mail ignorieren."},
		"test":    {"Testmail von {site}", "Hallo {name},\n\ndas ist eine Testmail von {site}. Mails funktionieren."},
	},
}

// Compose fills in one kind of mail in the recipient's language.
func Compose(kind, locale, to string, vars map[string]string) Message {
	lang := texts[strings.SplitN(locale, "-", 2)[0]]
	t, ok := lang[kind]
	if !ok {
		t = texts["en"][kind]
	}
	fill := func(s string) string {
		for k, v := range vars {
			s = strings.ReplaceAll(s, "{"+k+"}", v)
		}
		return strings.NewReplacer("Hello ,", "Hello,", "Hallo ,", "Hallo,").Replace(s)
	}
	subject := strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return ' '
		}
		return r
	}, fill(t[0]))
	return Message{To: to, Subject: subject, Body: fill(t[1])}
}

// Kinds lists every kind of mail, for tests.
func Kinds() []string { return []string{"invite", "setup", "reset", "confirm", "test"} }
