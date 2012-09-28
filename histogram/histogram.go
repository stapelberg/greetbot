// vim:ts=4:sw=4
// i3 - IRC bot to greet people and tell them to wait
// © 2012 Michael Stapelberg (see also: LICENSE)
package histogram

import (
	"fmt"
	"log"
	"encoding/gob"
	"time"
	"os"
	"strings"
	"syscall"
)

// A combination of time.Weekday and an integer representing each hour of the
// day. This type is used as the key for our histogram.
type WeekdayHour struct {
	Weekday time.Weekday
	Hour int
}

type histogramEntry struct {
	// The average delay between two messages, in seconds.
	AverageDelayBetweenMessages int64

	// The number of different IRC nicknames counted during this particular
	// hour.
	NumberOfNicknames int
}

type currentStats struct {
	// Stores all the different nicknames.
	nicknames map[string]bool

	// The following two fields form the average delay between messages, which
	// roughly indicates how active the channel is.
	delayBetweenMessages int64
	numberOfMessages int64
	lastMessage time.Time

	// The weekday/hour this struct is for. If the current weekday/hour doesn’t
	// match this one anymore, we need to store this struct and begin a new
	// one.
	weekdayHour WeekdayHour
}

type Histogram struct {
	filename string
	current currentStats
	Histogram map[WeekdayHour]histogramEntry
}

func (h *Histogram) IsActive() bool {
	fmt.Printf("Checking if the channel is currently inactive.\n")
	// TODO: we probably want the median here, then compare the current and last-hour values
	return true
}

func (h *Histogram) LogActivity(nickname string) {
	now := time.Now().UTC()
	key := WeekdayHour{now.Weekday(), now.Hour()}
	if h.current.weekdayHour != key {
		fmt.Printf("Rotating, current = %s, key = %s\n", h.current.weekdayHour, key)
		entry := h.Histogram[key]
		if h.current.numberOfMessages > 0 {
			entry.AverageDelayBetweenMessages =
				(h.current.delayBetweenMessages / h.current.numberOfMessages)
		} else {
			entry.AverageDelayBetweenMessages = -1
		}
		entry.NumberOfNicknames = len(h.current.nicknames)
		h.Histogram[key] = entry

		h.current.delayBetweenMessages = 0
		h.current.numberOfMessages = 0
		h.current.nicknames = make(map[string]bool, 100)
		h.current.weekdayHour = key
	}

	h.current.delayBetweenMessages += (now.Unix() - h.current.lastMessage.Unix())
	h.current.numberOfMessages += 1
	h.current.nicknames[strings.ToLower(nickname)] = true

	fmt.Println(h)

	// Write the stats file to disk. We might not want to do this on every change.
	file, err := os.OpenFile(h.filename, os.O_WRONLY | os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(`Could not open histogram file "%s": %s`, h.filename, err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(h); err != nil {
		log.Fatal(`Could not write histogram data to "%s": %s`, h.filename, err)
	}
}

func Load(filename string) Histogram {
	now := time.Now().UTC()
	var result Histogram
	result.filename = filename
	result.Histogram = make(map[WeekdayHour]histogramEntry, 0)

	// Check if the file exists and load it, if so. Otherwise, create a new file.
	if _, err := os.Stat(filename); err != nil {
		if e, ok := err.(*os.PathError); ok && e.Err != syscall.ENOENT {
			log.Fatalf(`Error loading histogram data from "%s": %s`, filename, err)
		}

		// Err == os.ENOENT, this is ok.
	} else {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf(`Error reading histogram data from "%s": %s`, filename, err)
		}
		defer file.Close()

		decoder := gob.NewDecoder(file)
		if err := decoder.Decode(&result); err != nil {
			log.Fatalf(`Could not load histogram from "%s": %s`, filename, err)
		}
	}

	// Initialize currentStats, no matter whether we loaded a previous state
	// file or not.
	result.current.nicknames = make(map[string]bool, 100)
	result.current.weekdayHour = WeekdayHour{now.Weekday(), now.Hour()}
	result.current.lastMessage = now

	return result
}
