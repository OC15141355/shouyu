package greeting

import (
	"strings"
	"testing"
	"time"
)

func TestGreetTimeOfDay(t *testing.T) {
	loc, _ := time.LoadLocation("Australia/Sydney")
	cases := []struct {
		hour int
		want string
	}{
		{5, "Morning"},
		{11, "Morning"},
		{12, "Afternoon"},
		{17, "Evening"},
		{22, "Evening"},
		{2, "Late night"},
	}
	for _, c := range cases {
		at := time.Date(2026, 5, 13, c.hour, 0, 0, 0, loc) // Wednesday
		got := Greet("declan", at)
		if !strings.HasPrefix(got, c.want) {
			t.Fatalf("hour %d: got %q, want prefix %q", c.hour, got, c.want)
		}
	}
}

func TestGreetFridaySpecial(t *testing.T) {
	loc, _ := time.LoadLocation("Australia/Sydney")
	friday := time.Date(2026, 5, 15, 10, 0, 0, 0, loc)
	got := Greet("declan", friday)
	if !strings.Contains(got, "Happy Friday") {
		t.Fatalf("friday morning didn't surface Happy Friday: %q", got)
	}
}

func TestGreetWeekendSpecial(t *testing.T) {
	loc, _ := time.LoadLocation("Australia/Sydney")
	sat := time.Date(2026, 5, 16, 10, 0, 0, 0, loc)
	got := Greet("declan", sat)
	if !strings.Contains(got, "weekend") && !strings.Contains(got, "Saturday") {
		t.Fatalf("Saturday didn't surface weekend/Saturday: %q", got)
	}
}

func TestGreetIncludesUsername(t *testing.T) {
	at := time.Now()
	if !strings.Contains(Greet("declan", at), "declan") {
		t.Fatal("username missing")
	}
}
