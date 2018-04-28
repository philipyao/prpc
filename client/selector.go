package client

import (
	"math/rand"
	"time"
)

type selector func(endPoints []*endPoint) *endPoint
type fnSelect func(config configSelect) selector

func init() {
	rand.Seed(time.Now().Unix())
}

func selectRandom(config configSelect) selector {
	return func(endPoints []*endPoint) *endPoint {
		if len(endPoints) == 0 {
			return nil
		}
		return endPoints[rand.Intn(len(endPoints))]
	}
}

func selectWeightedRandom(config configSelect) selector {
	return func(endPoints []*endPoint) *endPoint {
		total := 0
		for _, ep := range endPoints {
			total += ep.weight
		}
		if total <= 0 {
			return nil
		}
		rd := rand.Intn(total)
		count := 0
		for _, ep := range endPoints {
			count += ep.weight
			if rd >= count {
				continue
			}
			return ep
		}
		return nil
	}
}

func selectRoundRobin(config configSelect) selector {
	var i int
	return func(endPoints []*endPoint) *endPoint {
		if len(endPoints) == 0 {
			return nil
		}
		ep := endPoints[i % len(endPoints)]
		i++
		return ep
	}
}

func selectSpecified(config configSelect) selector {
	return func(endPoints []*endPoint) *endPoint {
		if config.index < 0 || config.index >= len(endPoints) {
			return nil
		}
		return endPoints[config.index]
	}
}

type selectType	int
const (
	SelectTypeRandom  selectType 	= iota
	SelectTypeWeightedRandom
	SelectTypeRoundRobin
	SelectTypeSpecified
)

var selectors = []struct{
	typ selectType
	desc string
	fn fnSelect
} {
	{ SelectTypeRandom, "SelectTypeRandom", selectRandom },
	{ SelectTypeWeightedRandom, "SelectTypeWeightedRandom", selectWeightedRandom },
	{ SelectTypeRoundRobin, "SelectTypeRoundRobin", selectRandom },
	{ SelectTypeSpecified, "SelectTypeSpecified", selectSpecified },
}

func createSelector(config configSelect) selector {
	if config.typ < 0 || config.typ >= len(selectors) {
		return nil
	}
	return selectors[config.typ].fn(config)
}

