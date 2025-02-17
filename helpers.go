package main

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

func printWithTimestamp(a ...any) {
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	message := fmt.Sprintf("%s %v", timestamp, a)
	formatted := strings.ReplaceAll(strings.ReplaceAll(message, "[", ""), "]", "")
	fmt.Println(formatted)
}

func trim(s string) string {
	return strings.Trim(s, " \n\r")
}

func hasAlphabet(input string) bool {
	hasAlphabetPattern := regexp.MustCompile("[a-zA-Z]")
	return hasAlphabetPattern.MatchString(input)
}

func toSlotSlice(sr []SlotRecord) SlotSlice {
	result := SlotSlice{}
	for _, record := range sr {
		sets := SetSlice{}

		for i := 1; i <= 5; i++ {
			idField := fmt.Sprintf("Set%dID", i)
			gamesField := fmt.Sprintf("Set%dGames", i)
			tiebreakField := fmt.Sprintf("Set%dTiebreak", i)

			idValue := reflect.ValueOf(record).FieldByName(idField)
			gamesValue := reflect.ValueOf(record).FieldByName(gamesField)
			tiebreakValue := reflect.ValueOf(record).FieldByName(tiebreakField)

			if gamesValue.IsValid() && !gamesValue.IsNil() {
				sets.add(Set{
					ID:         idValue.String(),
					DrawSlotID: record.ID,
					Number:     i,
					Games:      *gamesValue.Interface().(*int),
					Tiebreak:   *tiebreakValue.Interface().(*int),
				})
			} else {
				break
			}
		}

		result.add(Slot{
			ID:       record.ID,
			DrawID:   record.DrawID,
			Round:    record.Round,
			Position: record.Position,
			Name:     record.Name,
			Seed:     record.Seed,
			Sets:     sets,
		})
	}
	return result
}

func getSlotKey(s Slot) string {
	formattedPosition := fmt.Sprintf("%03d", s.Position)
	return fmt.Sprintf("%d.%s", s.Round, formattedPosition)
}

func getUpdates(scraped SlotSlice, current SlotSlice, seeds map[string]string) (SlotSlice, SlotSlice, SetSlice, SetSlice) {
	scrapedMap := make(map[string]Slot)
	currentMap := make(map[string]Slot)
	allKeys := make(map[string]bool)

	for _, slot := range scraped {
		key := getSlotKey(slot)
		scrapedMap[key] = slot
		allKeys[key] = true
	}

	for _, slot := range current {
		key := getSlotKey(slot)
		currentMap[key] = slot
		allKeys[key] = true
	}

	keys := []string{}
	for k := range allKeys {
		keys = append(keys, k)
	}

	// Sort keys to ensure consistent order for testing/debugging
	sort.Strings(keys)

	newSlots := SlotSlice{}
	updatedSlots := SlotSlice{}
	newSets := SetSlice{}
	updatedSets := SetSlice{}

	for _, key := range keys {
		scrapedSlot, scrapedExists := scrapedMap[key]
		currentSlot, currentExists := currentMap[key]

		// New slot
		if !currentExists {
			newSlots.add(scrapedSlot)
			for _, set := range scrapedSlot.Sets {
				newSets.add(Set{
					DrawSlotID: scrapedSlot.ID,
					Number:     set.Number,
					Games:      set.Games,
					Tiebreak:   set.Tiebreak,
				})
			}
			continue
		}

		// Existing slot isn't scraped
		if !scrapedExists {
			continue
		}

		// Update set scores
		// Scraped slots and sets don't have IDs so we can't compare them directly
		for j, scrapedSet := range scrapedSlot.Sets {
			if j < len(currentSlot.Sets) {
				currentSet := currentSlot.Sets[j]

				// Current and scraped sets are in order on each slot so set numbers should match
				if currentSet.Number != scrapedSet.Number {
					log.Println("Set numbers don't match for Set with ID:", currentSet.ID, "Current:", currentSet.Number, "Scraped:", scrapedSet.Number)
					continue
				}

				if currentSet.Games != scrapedSet.Games || currentSet.Tiebreak != scrapedSet.Tiebreak {
					updatedSets.add(Set{
						ID:         currentSet.ID,
						DrawSlotID: currentSlot.ID,
						Number:     scrapedSet.Number,
						Games:      scrapedSet.Games,
						Tiebreak:   scrapedSet.Tiebreak,
					})
				}
			} else {
				// Add new set score
				newSets.add(Set{
					DrawSlotID: currentSlot.ID,
					Number:     scrapedSet.Number,
					Games:      scrapedSet.Games,
					Tiebreak:   scrapedSet.Tiebreak,
				})
			}
		}

		newName := scrapedSlot.Name
		newSeed := seeds[newName]

		// Don't clear slots with existing name
		if newName == "" {
			continue
		}

		// No update needed
		if newName == currentSlot.Name && newSeed == currentSlot.Seed {
			continue
		}

		updatedSlot := Slot{
			ID:       currentSlot.ID,
			DrawID:   currentSlot.DrawID,
			Round:    currentSlot.Round,
			Position: currentSlot.Position,
			Name:     newName,
			Seed:     newSeed,
			Sets:     scrapedSlot.Sets,
		}

		updatedSlots.add(updatedSlot)
	}

	return newSlots, updatedSlots, newSets, updatedSets
}

// Example usage:
// err:= saveHTMLToFile(html, "scraped_pages/wtaRendered.html")
//
//	if err != nil {
//	  log.Println("Error saving HTML to file:", err)
//	}
func saveHTMLToFile(html, filename string) error {
	return os.WriteFile(filename, []byte(html), 0644)
}

func readHTMLFromFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
