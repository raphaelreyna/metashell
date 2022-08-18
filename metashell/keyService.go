package metashell

import (
	"crypto/md5"
	"fmt"
	"math"
	"sync"

	"github.com/agnivade/levenshtein"
)

type vector struct {
	tty       string
	command   string
	timestamp int64
}

func vectorMetric(v1, v2 *vector) float64 {
	if v1.tty != v2.tty {
		return math.MaxFloat64
	}

	dt := v1.timestamp - v2.timestamp
	if dt < 0 {
		dt *= -1
	}

	dc := int64(levenshtein.ComputeDistance(v1.command, v2.command))

	return math.Sqrt(float64(dt*dt + dc*dc))
}

func (v *vector) key() string {
	h := md5.Sum([]byte(v.tty + v.command))
	return fmt.Sprintf("%d-%x", v.timestamp, h)
}

type cmdKeyService struct {
	assignedKeys map[string]*vector
	timeSeries   []*vector
	sync.RWMutex
}

func (cks *cmdKeyService) registerVector(v *vector) string {
	cks.Lock()
	defer cks.Unlock()

	if cks.assignedKeys == nil {
		cks.assignedKeys = map[string]*vector{}
	}
	if cks.timeSeries == nil {
		cks.timeSeries = []*vector{}
	}

	key := v.key()
	cks.assignedKeys[key] = v
	cks.timeSeries = append(cks.timeSeries, v)

	return key
}

func (cks *cmdKeyService) getKey(v *vector) string {
	cks.Lock()
	defer cks.Unlock()

	if cks.assignedKeys == nil || cks.timeSeries == nil {
		return ""
	}

	// find the closest match
	var (
		minv *vector
		min  = math.MaxFloat64
	)
	for idx := len(cks.timeSeries) - 1; 0 <= idx; idx-- {
		vv := cks.timeSeries[idx]

		d := vectorMetric(v, vv)
		switch {
		case d < min:
			min = d
			minv = vv
		case d == min:
			// use time as the tiebreaker
			dt := vv.timestamp - v.timestamp
			if dt < 0 {
				dt *= -1
			}

			minDt := minv.timestamp - v.timestamp
			if minDt < 0 {
				minDt *= -1
			}

			if dt < minDt {
				min = d
				minv = vv
			}
		}
	}

	// find the key
	var key string
	for k, v := range cks.assignedKeys {
		if v == minv {
			key = k
			break
		}
	}

	return key
}

func (cks *cmdKeyService) exchangeKey(k string) *vector {
	cks.Lock()
	defer cks.Unlock()

	if len(cks.assignedKeys) == 0 || len(cks.timeSeries) == 0 {
		return nil
	}

	v, ok := cks.assignedKeys[k]
	if !ok {
		return nil
	}

	delete(cks.assignedKeys, k)
	idx := -1
	for i := 0; i < len(cks.timeSeries); i++ {
		if cks.timeSeries[i] == v {
			idx = i
			break
		}
	}
	if 0 <= idx {
		cks.timeSeries = append(cks.timeSeries[:idx], cks.timeSeries[idx+1:]...)
	}

	return v
}
