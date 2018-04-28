package client

import (
	"math/rand"
	"time"
	"fmt"
	"errors"
)

type selector func(endPoints []*endPoint) *endPoint
type fnSelect func(config configSelect) (selector, error)

func init() {
	rand.Seed(time.Now().Unix())
}

func selectRandom(config configSelect) (selector, error) {
	return func(endPoints []*endPoint) *endPoint {
		if len(endPoints) == 0 {
			return nil
		}
		return endPoints[rand.Intn(len(endPoints))]
	}, nil
}

func selectWeightedRandom(config configSelect) (selector, error) {
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
	}, nil
}

func selectRoundRobin(config configSelect) (selector, error) {
	var i int
	return func(endPoints []*endPoint) *endPoint {
		if len(endPoints) == 0 {
			return nil
		}
		ep := endPoints[i % len(endPoints)]
		i++
		return ep
	}, nil
}

func selectSpecified(config configSelect) (selector, error) {
	if config.index < 0 {
		return nil, errors.New("index not specified")
	}
	return func(endPoints []*endPoint) *endPoint {
		if config.index >= len(endPoints) {
			return nil
		}
		return endPoints[config.index]
	}, nil
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
	{ SelectTypeRoundRobin, "SelectTypeRoundRobin", selectRoundRobin },
	{ SelectTypeSpecified, "SelectTypeSpecified", selectSpecified },
}

func createSelector(config configSelect) (selector, error) {
	itype := int(config.typ)
	if itype < 0 || itype >= len(selectors) {
		return nil, fmt.Errorf("unsupported select type %v", config.typ)
	}
	return selectors[config.typ].fn(config)
}

