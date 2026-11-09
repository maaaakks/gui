package main

import "testing"

func TestStoryDifficultyDoesNotResetHandicapsEachChapter(t *testing.T) {
	for _, format := range []string{"solo", "duo", "trio", "ascension"} {
		s := newStorySave()
		s.Format = format
		s.Ranks = map[string]int{"1v1": 16, "2v2": 16, "3v3": 16}
		lastCap, lastRank := float32(0), 0
		for chapter, city := range storyCitiesForSave(s) {
			for _, c := range city.Challenges {
				if c.OpponentBoostCap < 70 || c.OpponentBoostCap > 100 || c.OpponentBoostCap < lastCap {
					t.Fatal("Opponent handicap resets or exceeds normal tank", format, c.ID, c.OpponentBoostCap)
				}
				if c.EnemyRank < lastRank || c.EnemyRank > 18 {
					t.Fatal("Difficulty drops or exceeds two reference divisions", format, c.ID, c.EnemyRank)
				}
				if chapter >= 3 && c.OpponentBoostCap != 100 {
					t.Fatal("Late opponents should have normal boost", c.ID)
				}
				lastCap, lastRank = c.OpponentBoostCap, c.EnemyRank
			}
		}
	}
}
