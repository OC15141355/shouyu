package greeting

import (
	"fmt"
	"time"
)

// Greet returns a greeting like "Morning, declan" / "Happy Friday, declan" /
// "Happy weekend, declan". Time-of-day base + day-of-week sprinkle on
// Friday/Saturday/Sunday.
func Greet(username string, at time.Time) string {
	base := timeOfDay(at)
	switch at.Weekday() {
	case time.Friday:
		return fmt.Sprintf("%s, %s — Happy Friday", base, username)
	case time.Saturday, time.Sunday:
		return fmt.Sprintf("%s, %s — Happy weekend", base, username)
	}
	return fmt.Sprintf("%s, %s", base, username)
}

func timeOfDay(at time.Time) string {
	switch h := at.Hour(); {
	case h >= 5 && h < 12:
		return "Morning"
	case h >= 12 && h < 17:
		return "Afternoon"
	case h >= 17 && h < 23:
		return "Evening"
	default:
		return "Late night"
	}
}
