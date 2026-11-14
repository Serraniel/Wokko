package knowledge

// These types are what the API returns. They're also generated for the frontend, so this file
// holds nothing else.

type Source struct {
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
	Added string `json:"added,omitempty"`
}

// Finding is one thing Wokko noticed, in the viewer's language.
type Finding struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Parent   string   `json:"parent,omitempty"`
	Severity string   `json:"severity,omitempty"`
	Title    string   `json:"title"`
	Text     string   `json:"text"`
	Tips     []string `json:"tips"`
	Steps    []int    `json:"steps"` // 1-based step numbers where it applies, when known
	Sources  []Source `json:"sources"`
}

type FoundAllergen struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Foods []string `json:"foods"`
	// Maybe means it only depends on the product, such as celery in stock cubes. Check the label.
	Maybe bool `json:"maybe"`
}

// Diet is true or false when Wokko knows, and nil when some food couldn't be classified.
type Diet struct {
	Vegetarian *bool    `json:"vegetarian"`
	Vegan      *bool    `json:"vegan"`
	Because    []string `json:"because"` // foods that rule it out
}

type Result struct {
	Sauces       []Finding       `json:"sauces"`
	Techniques   []Finding       `json:"techniques"`
	Problems     []Finding       `json:"problems"`
	Safety       []Finding       `json:"safety"`
	Allergens    []FoundAllergen `json:"allergens"`
	Diet         Diet            `json:"diet"`
	Unclassified []string        `json:"unclassified"` // foods Wokko knows nothing about
}
