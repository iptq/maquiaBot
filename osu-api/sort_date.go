package osuapi

import (
	"sort"
	"time"
)

// Sorts the slice in place
func DateSortGUS(x []GUSScore) (err error) {
	sort.Slice(x, func(i, j int) bool {
		var time1, time2 time.Time

		time1, err = time.Parse("2006-01-02 15:04:05", x[i].Date.String())
		if err != nil {
			return false
		}

		time2, err = time.Parse("2006-01-02 15:04:05", x[j].Date.String())
		if err != nil {
			return false
		}

		return time1.Unix() > time2.Unix()
	})
	return
}
