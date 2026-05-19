package scheduler

import (
	"fmt"
	"strings"
	"time"
)

// cronSchedule represents a parsed cron expression.
type cronSchedule struct {
	minute  *cronField
	hour    *cronField
	day     *cronField
	month   *cronField
	weekday *cronField
}

// cronField represents a single field in a cron expression.
type cronField struct {
	values map[int]bool
	any    bool
}

// parseCron parses a cron expression string into a cronSchedule.
// The expression must be in the format: "minute hour day month weekday"
func parseCron(expression string) (*cronSchedule, error) {
	parts := strings.Fields(expression)
	if len(parts) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields, got %d", len(parts))
	}

	minute, err := parseField(parts[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("minute field: %w", err)
	}

	hour, err := parseField(parts[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("hour field: %w", err)
	}

	day, err := parseField(parts[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("day field: %w", err)
	}

	month, err := parseField(parts[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("month field: %w", err)
	}

	weekday, err := parseField(parts[4], 0, 7)
	if err != nil {
		return nil, fmt.Errorf("weekday field: %w", err)
	}

	return &cronSchedule{
		minute:  minute,
		hour:    hour,
		day:     day,
		month:   month,
		weekday: weekday,
	}, nil
}

// parseField parses a single cron field (e.g., "0", "*/5", "1-5", "1,3,5").
func parseField(field string, min, max int) (*cronField, error) {
	if field == "*" {
		return &cronField{any: true}, nil
	}

	cf := &cronField{
		values: make(map[int]bool),
	}

	// Handle step values (e.g., "*/5", "1-10/2")
	if strings.Contains(field, "/") {
		parts := strings.Split(field, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid step format: %s", field)
		}

		var step int
		if _, err := fmt.Sscanf(parts[1], "%d", &step); err != nil {
			return nil, fmt.Errorf("invalid step value: %s", parts[1])
		}
		if step <= 0 {
			return nil, fmt.Errorf("step must be positive: %d", step)
		}

		rangePart := parts[0]
		if rangePart == "*" {
			// */5 means every 5th value
			for i := min; i <= max; i += step {
				cf.values[i] = true
			}
		} else if strings.Contains(rangePart, "-") {
			// 1-10/2 means every 2nd value from 1 to 10
			rangeParts := strings.Split(rangePart, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", rangePart)
			}

			var start, end int
			if _, err := fmt.Sscanf(rangeParts[0], "%d", &start); err != nil {
				return nil, fmt.Errorf("invalid range start: %s", rangeParts[0])
			}
			if _, err := fmt.Sscanf(rangeParts[1], "%d", &end); err != nil {
				return nil, fmt.Errorf("invalid range end: %s", rangeParts[1])
			}

			if start < min || start > max || end < min || end > max || start > end {
				return nil, fmt.Errorf("range %d-%d out of bounds [%d-%d]", start, end, min, max)
			}

			for i := start; i <= end; i += step {
				cf.values[i] = true
			}
		} else {
			// Single value with step (e.g., "5/10" - not standard but handle it)
			var start int
			if _, err := fmt.Sscanf(rangePart, "%d", &start); err != nil {
				return nil, fmt.Errorf("invalid value: %s", rangePart)
			}
			if start < min || start > max {
				return nil, fmt.Errorf("value %d out of bounds [%d-%d]", start, min, max)
			}

			for i := start; i <= max; i += step {
				cf.values[i] = true
			}
		}

		return cf, nil
	}

	// Handle ranges (e.g., "1-5")
	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid range format: %s", field)
		}

		var start, end int
		if _, err := fmt.Sscanf(parts[0], "%d", &start); err != nil {
			return nil, fmt.Errorf("invalid range start: %s", parts[0])
		}
		if _, err := fmt.Sscanf(parts[1], "%d", &end); err != nil {
			return nil, fmt.Errorf("invalid range end: %s", parts[1])
		}

		if start < min || start > max || end < min || end > max || start > end {
			return nil, fmt.Errorf("range %d-%d out of bounds [%d-%d]", start, end, min, max)
		}

		for i := start; i <= end; i++ {
			cf.values[i] = true
		}

		return cf, nil
	}

	// Handle lists (e.g., "1,3,5")
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		for _, part := range parts {
			var value int
			if _, err := fmt.Sscanf(strings.TrimSpace(part), "%d", &value); err != nil {
				return nil, fmt.Errorf("invalid list value: %s", part)
			}
			if value < min || value > max {
				return nil, fmt.Errorf("value %d out of bounds [%d-%d]", value, min, max)
			}
			cf.values[value] = true
		}

		return cf, nil
	}

	// Handle single value
	var value int
	if _, err := fmt.Sscanf(field, "%d", &value); err != nil {
		return nil, fmt.Errorf("invalid value: %s", field)
	}
	if value < min || value > max {
		return nil, fmt.Errorf("value %d out of bounds [%d-%d]", value, min, max)
	}

	cf.values[value] = true
	return cf, nil
}

// matches checks if a value matches the cron field.
func (cf *cronField) matches(value int) bool {
	if cf.any {
		return true
	}
	return cf.values[value]
}

// next calculates the next execution time from the given time.
func (cs *cronSchedule) next(from time.Time) time.Time {
	// Start from the next minute
	current := from.Truncate(time.Minute).Add(time.Minute)

	// Limit search to avoid infinite loops (max 1 year ahead)
	maxAttempts := 365 * 24 * 60
	attempts := 0

	for attempts < maxAttempts {
		attempts++

		// Check month
		if !cs.month.matches(int(current.Month())) {
			// Move to first day of next month
			current = time.Date(current.Year(), current.Month()+1, 1, 0, 0, 0, 0, current.Location())
			continue
		}

		// Check day of month and weekday
		// Cron allows both day-of-month and weekday, and if both are specified,
		// it means "either condition" (OR logic)
		dayMatches := cs.day.matches(current.Day())
		weekdayMatches := cs.weekday.matches(int(current.Weekday()))

		// If both are specified (not wildcard), use OR logic
		// If one is wildcard, use the other
		if !cs.day.any && !cs.weekday.any {
			// Both specified: match if either matches
			if !dayMatches && !weekdayMatches {
				current = current.AddDate(0, 0, 1)
				current = time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location())
				continue
			}
		} else if !cs.day.any {
			// Only day specified
			if !dayMatches {
				current = current.AddDate(0, 0, 1)
				current = time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location())
				continue
			}
		} else if !cs.weekday.any {
			// Only weekday specified
			if !weekdayMatches {
				current = current.AddDate(0, 0, 1)
				current = time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location())
				continue
			}
		}

		// Check hour
		if !cs.hour.matches(current.Hour()) {
			current = current.Add(time.Hour)
			current = time.Date(current.Year(), current.Month(), current.Day(), current.Hour(), 0, 0, 0, current.Location())
			continue
		}

		// Check minute
		if !cs.minute.matches(current.Minute()) {
			current = current.Add(time.Minute)
			continue
		}

		// All fields match
		return current
	}

	// Could not find next execution time (should not happen in practice)
	return time.Time{}
}
