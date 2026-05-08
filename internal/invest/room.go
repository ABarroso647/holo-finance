package invest

import "time"

const (
	TFSAAnnualLimit = 7000.0
	FHSAAnnualLimit = 8000.0
	FHSALifetimeMax = 40000.0
)

// jansPassed counts how many January 1sts have occurred strictly after referenceDate
// and on or before now.
func jansPassed(referenceDate, now time.Time) int {
	count := 0
	for y := referenceDate.Year() + 1; y <= now.Year(); y++ {
		jan1 := time.Date(y, time.January, 1, 0, 0, 0, 0, time.UTC)
		if !jan1.After(now) && jan1.After(referenceDate) {
			count++
		}
	}
	return count
}

// CurrentTFSARoom computes current TFSA room given the room at a reference date.
// Adds $7,000 for each Jan 1 that has passed since referenceDate.
func CurrentTFSARoom(roomAtDate float64, referenceDate time.Time, now time.Time) float64 {
	if referenceDate.IsZero() {
		referenceDate = now
	}
	years := jansPassed(referenceDate, now)
	return roomAtDate + float64(years)*TFSAAnnualLimit
}

// CurrentFHSARoom computes current FHSA room given the room at a reference date.
// Adds $8,000 for each Jan 1 that has passed since referenceDate, capped at FHSALifetimeMax.
func CurrentFHSARoom(roomAtDate float64, referenceDate time.Time, now time.Time) float64 {
	if referenceDate.IsZero() {
		referenceDate = now
	}
	years := jansPassed(referenceDate, now)
	room := roomAtDate + float64(years)*FHSAAnnualLimit
	if room > FHSALifetimeMax {
		room = FHSALifetimeMax
	}
	return room
}
