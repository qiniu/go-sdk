package freezer

import (
	"sync"
	"time"
)

type Freezer interface {
	Available(itemId string) bool
	Freeze(itemId string, duration int64) error
	Unfreeze(itemId string) error
}

func New() Freezer {
	return &freezer{
		freezerItems: &sync.Map{},
	}
}

type freezer struct {
	freezerItems *sync.Map
}

func (i *freezer) Available(itemId string) bool {
	unfreezeTime, ok := i.freezerItems.Load(itemId)
	if !ok {
		return true
	}

	unfreezeTimeInt64, ok := unfreezeTime.(int64)
	if !ok {
		return false
	}

	timestamp := time.Now().Unix()
	return timestamp > unfreezeTimeInt64
}

func (i *freezer) Freeze(itemId string, duration int64) error {
	timestamp := time.Now().Unix()
	i.freezerItems.Store(itemId, timestamp+duration)
	return nil
}

func (i *freezer) Unfreeze(itemId string) error {
	i.freezerItems.Delete(itemId)
	return nil
}
