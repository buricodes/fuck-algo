package domain

import "slices"

const (
	LicenceTrial  = "trial"
	LicenceActive = "active"
)

type Licence struct {
	Status string `json:"status"`
	Trial  Trial  `json:"trial"`
}

type Trial struct {
	VoiceLeft  int `json:"voiceLeft"`
	ChecksLeft int `json:"checksLeft"`
}

type SpoilerType string

const (
	SpoilerAnime  SpoilerType = "anime"
	SpoilerGame   SpoilerType = "game"
	SpoilerShow   SpoilerType = "show"
	SpoilerMovie  SpoilerType = "movie"
	SpoilerSports SpoilerType = "sports"
)

var SpoilerTypes = []SpoilerType{SpoilerAnime, SpoilerGame, SpoilerShow, SpoilerMovie, SpoilerSports}

func (t SpoilerType) Valid() bool { return slices.Contains(SpoilerTypes, t) }

type Strictness int

const (
	Relaxed  Strictness = 0
	Balanced Strictness = 1
	Paranoid Strictness = 2
)

func (s Strictness) Valid() bool { return s >= Relaxed && s <= Paranoid }

type KeywordAction string

var KeywordActions = []KeywordAction{"hide", "blur", "skip"}

func (a KeywordAction) Valid() bool { return slices.Contains(KeywordActions, a) }

type Site string

var Sites = []Site{"youtube", "x", "instagram", "web"}

func (s Site) Valid() bool { return slices.Contains(Sites, s) }

type WatchContext struct {
	Name     string      `json:"name"`
	Type     SpoilerType `json:"type"`
	Level    Strictness  `json:"level"`
	Keywords []string    `json:"keywords"`
}

type ProposedRule interface{ ruleKind() string }

type SpoilerRule struct {
	Kind     string      `json:"kind"`
	Name     string      `json:"name"`
	Type     SpoilerType `json:"type"`
	Level    Strictness  `json:"level"`
	Keywords []string    `json:"keywords"`
}

type RepeatRule struct {
	Kind  string `json:"kind"`
	Topic string `json:"topic"`
	Sites []Site `json:"sites"`
}

type KeywordRule struct {
	Kind      string        `json:"kind"`
	Word      string        `json:"word"`
	Action    KeywordAction `json:"action"`
	ExpiresAt *string       `json:"expiresAt"`
}

func (SpoilerRule) ruleKind() string { return "spoiler" }
func (RepeatRule) ruleKind() string  { return "repeat" }
func (KeywordRule) ruleKind() string { return "keyword" }

type CheckItem struct {
	ID      string `json:"id"`
	Site    Site   `json:"site"`
	Title   string `json:"title"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

type CheckResult struct {
	ID        string  `json:"id"`
	SpoilerOf *string `json:"spoilerOf"`
	Topic     string  `json:"topic"`
}
