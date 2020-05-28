package fibbage

type User struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Score struct {
	ID    string `json:"id,omitempty"`
	Score int    `json:"score,omitempty"`
}

type Ready struct {
	ID    string `json:"id,omitempty"`
	Ready bool   `json:"ready,omitempty"`
}

type Categories struct {
	ID       string `json:"id,omitempty"`
	Category string `json:"category,omitempty"`
}

type Error struct {
	ID      string `json:"id,omitempty"`
	Message string `json:"error,omitempty"`
}

type Status struct {
	ID         string `json:"id,omitempty"`
	State      string `json:"state,omitempty"`
	ServerTime string `json:"time,omitempty"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
}

type Question struct {
	ID                 uint     `json:"id,omitempty"`
	Category           string   `json:"category"`
	Question           string   `json:"question"`
	Answer             string   `json:"answer"`
	AlternateSpellings []string `json:"alternateSpellings"`
	Suggestions        []string `json:"suggestions"`
}

type GameState struct {
	Mode     int       `json:"mode,omitempty"`
	State    int       `json:"state,omitempty"`
	Users    []User    `json:"users,omitempty"`
	Question *Question `json:"question,omitempty"`
}
