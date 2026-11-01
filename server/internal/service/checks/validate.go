package checks

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/buricodes/fuck-algo/server/internal/domain"
)

const (
	MaxItems        = 25
	maxIDChars      = 200
	maxTitleChars   = 300
	maxChannelChars = 200
	maxTextChars    = 600
	maxWatchlist    = 100
	maxNameChars    = 100
	maxKeywords     = 20
	maxKeywordChars = 60
	maxTopics       = 200
)

func Validate(req Request) error {
	if len(req.Items) == 0 || len(req.Items) > MaxItems {
		return fmt.Errorf("items must hold between 1 and %d entries", MaxItems)
	}
	seen := map[string]bool{}
	for i, item := range req.Items {
		if err := validateItem(item); err != nil {
			return fmt.Errorf("items[%d]: %w", i, err)
		}
		if seen[item.ID] {
			return fmt.Errorf("items[%d]: duplicate id %q", i, item.ID)
		}
		seen[item.ID] = true
	}
	if len(req.Watchlist) > maxWatchlist {
		return fmt.Errorf("watchlist may hold at most %d titles", maxWatchlist)
	}
	for i, w := range req.Watchlist {
		if err := validateWatch(w); err != nil {
			return fmt.Errorf("watchlist[%d]: %w", i, err)
		}
	}
	return validateStrings("topics", req.Topics, maxTopics, maxTopicChars)
}

func validateItem(item domain.CheckItem) error {
	switch {
	case item.ID == "" || utf8.RuneCountInString(item.ID) > maxIDChars:
		return fmt.Errorf("id must be 1 to %d characters", maxIDChars)
	case !item.Site.Valid():
		return errors.New("site must be youtube, x, instagram or web")
	case item.Title == "" && item.Text == "":
		return errors.New("title or text is required")
	case utf8.RuneCountInString(item.Title) > maxTitleChars:
		return fmt.Errorf("title may be at most %d characters", maxTitleChars)
	case utf8.RuneCountInString(item.Channel) > maxChannelChars:
		return fmt.Errorf("channel may be at most %d characters", maxChannelChars)
	case utf8.RuneCountInString(item.Text) > maxTextChars:
		return fmt.Errorf("text may be at most %d characters", maxTextChars)
	}
	return nil
}

func validateWatch(w domain.WatchContext) error {
	switch {
	case w.Name == "" || utf8.RuneCountInString(w.Name) > maxNameChars:
		return fmt.Errorf("name must be 1 to %d characters", maxNameChars)
	case !w.Type.Valid():
		return errors.New("type must be anime, game, show, movie or sports")
	case !w.Level.Valid():
		return errors.New("level must be 0, 1 or 2")
	}
	return validateStrings("keywords", w.Keywords, maxKeywords, maxKeywordChars)
}

func validateStrings(field string, values []string, maxCount, maxChars int) error {
	if len(values) > maxCount {
		return fmt.Errorf("%s may hold at most %d entries", field, maxCount)
	}
	for _, v := range values {
		if v == "" || utf8.RuneCountInString(v) > maxChars {
			return fmt.Errorf("each of %s must be 1 to %d characters", field, maxChars)
		}
	}
	return nil
}
