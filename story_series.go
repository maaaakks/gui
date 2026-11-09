package main

import (
	"errors"
	"fmt"
	"time"
)

// Series scores count game victories, independently of each game's goal score.
type StorySeries struct {
	Wins   int `json:"wins"`
	Losses int `json:"losses"`
}

func storyBestOf(c StoryChallenge) int {
	if c.BestOf == 3 || c.BestOf == 5 {
		return c.BestOf
	}
	return 1
}
func storySeriesOpen(c StoryChallenge, score StorySeries) bool {
	target := storyBestOf(c)/2 + 1
	return storyBestOf(c) > 1 && score.Wins+score.Losses > 0 && score.Wins < target && score.Losses < target
}
func storyHasOpenSeries(save StorySave) bool {
	for _, city := range storyCities() {
		for _, c := range city.Challenges {
			if storySeriesOpen(c, save.Series[c.ID]) {
				return true
			}
		}
	}
	return false
}

// Called once by the observer when a game finishes. Interrupted games never
// reach this function, so closing the game preserves the last completed score.
func storyRecordGame(save *StorySave, c StoryChallenge, blue, orange, received, demos, goals, skill int) string {
	passed := storyCompleted(c, blue, orange, received, demos, goals)
	attempt := StoryAttempt{KilobyteSkill: &skill, ChallengeID: c.ID, Won: blue > orange, Blue: blue, Orange: orange, At: time.Now().UTC().Format(time.RFC3339)}
	if storyBestOf(c) == 1 {
		attempt.Completed = passed
		save.Attempts = append(save.Attempts, attempt)
		storyResultReward(save, c, passed)
		if passed {
			return "Fixture complete. Reward saved."
		}
		return "Challenge failed. Try again."
	}
	if save.Series == nil {
		save.Series = map[string]StorySeries{}
	}
	score := save.Series[c.ID]
	target := storyBestOf(c)/2 + 1
	// A terminal result stays visible in the save until a new game is completed.
	// Retrying a lost series or replaying a won series starts at zero.
	if score.Wins >= target || score.Losses >= target {
		score = StorySeries{}
	}
	if passed {
		score.Wins++
	} else {
		score.Losses++
	}
	save.Series[c.ID] = score
	attempt.SeriesBestOf = storyBestOf(c)
	attempt.SeriesWins = score.Wins
	attempt.SeriesLosses = score.Losses
	attempt.Completed = score.Wins >= target
	save.Attempts = append(save.Attempts, attempt)
	if score.Wins >= target {
		storyResultReward(save, c, true)
		return fmt.Sprintf("BO%d won %d–%d. Reward saved.", storyBestOf(c), score.Wins, score.Losses)
	}
	if score.Losses >= target {
		storyResultReward(save, c, false)
		return fmt.Sprintf("BO%d lost %d–%d. Start a new series to try again.", storyBestOf(c), score.Wins, score.Losses)
	}
	return fmt.Sprintf("BO%d: %d–%d. Game %d is ready; series progress saved.", storyBestOf(c), score.Wins, score.Losses, score.Wins+score.Losses+1)
}

func (a *App) ResetStorySeries(id string) error {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	if a.storyStatus.Active {
		return errors.New("Finish or stop the current match first")
	}
	save, err := a.loadStory()
	if err != nil {
		return err
	}
	_, c, err := storyChapterMatch(save, id)
	if err != nil {
		return err
	}
	if !storySeriesOpen(c, save.Series[id]) {
		return errors.New("No unfinished series to restart")
	}
	delete(save.Series, id)
	if err = a.writeStory(save); err != nil {
		return err
	}
	a.storyStatus = StoryStatus{ChallengeID: id, Message: "Series reset to 0–0. Completed fixtures and credits are unchanged."}
	return nil
}
