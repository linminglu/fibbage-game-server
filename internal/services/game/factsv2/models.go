package factsv2

type (
	Message struct {
		CurrentPlayerId string                      `json:"currentPlayerId,omitempty"`
		Ticks           int                         `json:"ticks,omitempty"`
		State           string                      `json:"state,omitempty"`
		Answers         []string                    `json:"answers,omitempty"`
		Other           *Question                   `json:"otherQuestion,omitempty"`
		Score           map[string]int              `json:"score,omitempty"`
		Total           map[string]int              `json:"total,omitempty"`
		Choices         map[string]*AnswerMatrixRow `json:"answerMatrix,omitempty"`
	}

	Player struct {
		name              string
		question          *Question
		categories        []string
		categoryId        int
		totalScore        int
		answerLie         string
		shuffledAnswerIdx int
		answerTruthId     int
		iconName          string
		ready             bool
		used              bool
		current           bool
		connected         bool
	}

	AnswerMatrixRow struct {
		Text      string   `json:"text,omitempty"`
		PickedIds []string `json:"pickedIds,omitempty"`
	}
	UserReady struct {
		UID   string `json:"id,omitempty"`
		Ready bool   `json:"ready,omitempty"`
	}

	InputMessage struct {
		CategoryId int    `json:"categoryId,omitempty"`
		Answer     string `json:"answer,omitempty"`
		AnswerId   int    `json:"answerId,omitempty"`
	}
	// NicknameMessage represents a message that user sent
	NicknameMessage struct {
		Nickname  string `json:"nickname"`
		GroupUuid string `json:"uuid"`
	}

	// NewUser message will be received when new user join room
	User struct {
		UID      string `json:"id,omitempty"`
		Name     string `json:"name,omitempty"`
		Icon     string `json:"icon,omitempty"`
		IsPlayer bool   `json:"isPlayer,omitempty"`
	}

	// AllMembers contains all members uid
	AllMembers struct {
		Members []string `json:"members"`
	}

	// Response represents the result of joining room
	Response struct {
		Code   int    `json:"code"`
		Result string `json:"result"`
	}
	Question struct {
		Question          string `json:"question,omitempty"`
		Answer            string `json:"answer,omitempty"`
		ShuffledAnswerIdx int    `json:"shuffledIdx,omitempty"`
	}
)
