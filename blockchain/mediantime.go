// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"math"
	"sort"
	"sync"
	"time"

	"massnet.org/mass-wallet/logging"
)

const (
	maxAllowedOffsetSecs = 70 * 60 // 1 hour 10 minutes
	similarTimeSecs      = 5 * 60  // 5 minutes
)

var (
	maxMedianTimeEntries = 200
)

type MedianTimeSource interface {
	AdjustedTime() time.Time
	AddTimeSample(id string, timeVal time.Time)
	Offset() time.Duration
}

// int64Sorter implements sort.Interface to allow a slice of 64-bit integers to
// be sorted.
type int64Sorter []int64

// Len returns the number of 64-bit integers in the slice.  It is part of the
// sort.Interface implementation.
func (s int64Sorter) Len() int {
	return len(s)
}

// Swap swaps the 64-bit integers at the passed indices.  It is part of the
// sort.Interface implementation.
func (s int64Sorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less returns whether the 64-bit integer with index i should sort before the
// 64-bit integer with index j.  It is part of the sort.Interface
// implementation.
func (s int64Sorter) Less(i, j int) bool {
	return s[i] < s[j]
}

type medianTime struct {
	mtx                sync.Mutex
	knownIDs           map[string]struct{}
	offsets            []int64
	offsetSecs         int64
	invalidTimeChecked bool
}

// Ensure the medianTime type implements the MedianTimeSource interface.
var _ MedianTimeSource = (*medianTime)(nil)

// AdjustedTime returns the current time adjusted by the median time offset as
// calculated from the time samples added by AddTimeSample.
//
// This function is safe for concurrent access and is part of the
// MedianTimeSource interface implementation.
func (m *medianTime) AdjustedTime() time.Time {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	// Limit the adjusted time to 1 second precision.
	now := time.Unix(time.Now().Unix(), 0)
	return now.Add(time.Duration(m.offsetSecs) * time.Second)
}

// AddTimeSample adds a time sample that is used when determining the median
// time of the added samples.
//
// This function is safe for concurrent access and is part of the
// MedianTimeSource interface implementation.
func (m *medianTime) AddTimeSample(sourceID string, timeVal time.Time) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	// Don't add time data from the same source.
	if _, exists := m.knownIDs[sourceID]; exists {
		return
	}
	m.knownIDs[sourceID] = struct{}{}

	now := time.Unix(time.Now().Unix(), 0)
	offsetSecs := int64(timeVal.Sub(now).Seconds())
	numOffsets := len(m.offsets)
	if numOffsets == maxMedianTimeEntries && maxMedianTimeEntries > 0 {
		m.offsets = m.offsets[1:]
		numOffsets--
	}
	m.offsets = append(m.offsets, offsetSecs)
	numOffsets++

	// Sort the offsets so the median can be obtained as needed later.
	sortedOffsets := make([]int64, numOffsets)
	copy(sortedOffsets, m.offsets)
	sort.Sort(int64Sorter(sortedOffsets))

	offsetDuration := time.Duration(offsetSecs) * time.Second
	logging.CPrint(logging.DEBUG, "Added time sample", logging.LogFormat{
		"offsetDuartion": offsetDuration,
		"totalOffsets":   numOffsets,
	})

	if numOffsets < 5 || numOffsets&0x01 != 1 {
		return
	}

	// At this point the number of offsets in the list is odd, so the
	// middle value of the sorted offsets is the median.
	median := sortedOffsets[numOffsets/2]

	// Set the new offset when the median offset is within the allowed
	// offset range.
	if math.Abs(float64(median)) < maxAllowedOffsetSecs {
		m.offsetSecs = median
	} else {
		m.offsetSecs = 0

		if !m.invalidTimeChecked {
			m.invalidTimeChecked = true

			// Find if any time samples have a time that is close
			// to the local time.
			var remoteHasCloseTime bool
			for _, offset := range sortedOffsets {
				if math.Abs(float64(offset)) < similarTimeSecs {
					remoteHasCloseTime = true
					break
				}
			}

			// Warn if none of the time samples are close.
			if !remoteHasCloseTime {
				logging.CPrint(logging.WARN, "Please check your date and time "+
					"are correct!  mass will not work "+
					"properly with an invalid time", logging.LogFormat{
					"remteHasCloseTime": remoteHasCloseTime,
				})
			}
		}
	}

	medianDuration := time.Duration(m.offsetSecs) * time.Second
	logging.CPrint(logging.DEBUG, "New time offset", logging.LogFormat{
		"offset": medianDuration,
	})
}

// Offset returns the number of seconds to adjust the local clock based upon the
// median of the time samples added by AddTimeData.
//
// This function is safe for concurrent access and is part of the
// MedianTimeSource interface implementation.
func (m *medianTime) Offset() time.Duration {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	return time.Duration(m.offsetSecs) * time.Second
}

// NewMedianTime returns a new instance of concurrency-safe implementation of
// the MedianTimeSource interface.  The returned implementation contains the
// rules necessary for proper time handling in the chain consensus rules and
// expects the time samples to be added from the timestamp field of the version
// message received from remote peers that successfully connect and negotiate.
func NewMedianTime() MedianTimeSource {
	return &medianTime{
		knownIDs: make(map[string]struct{}),
		offsets:  make([]int64, 0, maxMedianTimeEntries),
	}
}
