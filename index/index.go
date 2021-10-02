package index

import (
	"strings"
	"sync"
)

type IndexCategory interface {
	Insert(field Field)
	Remove(field Field)
	Search(field Field, target ResultBuffer)
}

type Index struct {
	categories      map[string]IndexCategory
	categoriesMutex sync.RWMutex
}

func New() *Index {
	return &Index{
		categories: make(map[string]IndexCategory),
	}
}

func (i *Index) AddCategory(name string, value IndexCategory) {
	i.categoriesMutex.Lock()
	i.categories[name] = value
	i.categoriesMutex.Unlock()
}

func (i *Index) Insert(fields ...Field) {
	i.categoriesMutex.RLock()
	for _, field := range fields {
		category, ok := i.categories[field.Name]
		if !ok {
			continue
		}
		category.Insert(field)
	}
	i.categoriesMutex.RUnlock()
}

func (i *Index) Remove(fields ...Field) {
	i.categoriesMutex.RLock()
	for _, field := range fields {
		category, ok := i.categories[field.Name]
		if !ok {
			continue
		}
		category.Remove(field)
	}
	i.categoriesMutex.RUnlock()
}

func (i *Index) Search(buffer ResultBuffer, fields ...Field) {
	i.categoriesMutex.RLock()
	for _, field := range fields {
		category, ok := i.categories[field.Name]
		if !ok {
			continue
		}
		category.Search(field, buffer)
	}
	i.categoriesMutex.RUnlock()
}

type IntCapsule struct {
	value int32
	data  interface{}
}

type IntIndexCategory struct {
	capsules      []IntCapsule
	capsulesMutex sync.RWMutex
	Synchronized  bool
}

func (i *IntIndexCategory) Insert(field Field) {
	if field.Type != FTInt {
		return
	}

	if i.Synchronized {
		i.capsulesMutex.Lock()
		defer i.capsulesMutex.Unlock()
	}

	if len(i.capsules) == 0 {
		i.capsules = append(i.capsules, IntCapsule{value: field.Int1, data: field.Value})
		return
	}

	idx, _ := i.BinSearch(field.Int1)
	i.capsules = append(i.capsules, IntCapsule{value: field.Int1})
	copy(i.capsules[idx+1:], i.capsules[idx:])

	i.capsules[idx].data = field.Value
}

func (i *IntIndexCategory) Search(field Field, buffer ResultBuffer) {
	if field.Type != FTInt && field.Type != FTRange {
		return
	}

	if i.Synchronized {
		i.capsulesMutex.RLock()
		defer i.capsulesMutex.RUnlock()
	}

	switch field.Type {
	case FTInt:
		idx, ok := i.BinSearch(field.Int1)
		if ok {
			buffer.Add(i.capsules[idx].data)
		}
	case FTRange:
		start, _ := i.BinSearch(field.Int1)
		end, _ := i.BinSearch(field.Int2)
		for j := start; j < end; j++ {
			buffer.Add(i.capsules[j].data)
		}
	}
}

func (i *IntIndexCategory) Remove(field Field) {
	if field.Type != FTInt {
		return
	}

	if i.Synchronized {
		i.capsulesMutex.Lock()
		defer i.capsulesMutex.Unlock()
	}

	idx, _ := i.BinSearch(field.Int1)

	for idx < len(i.capsules) && i.capsules[idx].value == field.Int1 {
		if i.capsules[idx].data == field.Value {
			i.capsules = append(i.capsules[:idx], i.capsules[idx+1:]...)
			return
		}
		idx++
	}
}

func (i *IntIndexCategory) BinSearch(value int32) (int, bool) {
	if len(i.capsules) == 0 {
		return 0, false
	}

	low, high := 0, len(i.capsules)
	for low < high {
		mid := (low + high) / 2
		midValue := i.capsules[mid].value
		if midValue < value {
			low = mid + 1
		} else {
			high = mid
		}
	}

	return low, low < len(i.capsules) && i.capsules[low].value == value
}

type StringCapsule struct {
	value string
	data  interface{}
}

type StringIndexCategory struct {
	capsules      []StringCapsule
	capsulesMutex sync.RWMutex
	Synchronized  bool
}

func (i *StringIndexCategory) Insert(field Field) {
	if field.Type != FTString {
		return
	}

	if i.Synchronized {
		i.capsulesMutex.Lock()
		defer i.capsulesMutex.Unlock()
	}

	if len(i.capsules) == 0 {
		i.capsules = append(i.capsules, StringCapsule{value: field.String, data: field.Value})
		return
	}

	idx, _ := i.BinSearch(field.String)
	i.capsules = append(i.capsules, StringCapsule{value: field.String})
	copy(i.capsules[idx+1:], i.capsules[idx:])

	i.capsules[idx].data = field.Value
}

func (i *StringIndexCategory) Search(field Field, buffer ResultBuffer) {
	if field.Type != FTString && field.Type != FTExactString {
		return
	}

	if i.Synchronized {
		i.capsulesMutex.RLock()
		defer i.capsulesMutex.RUnlock()
	}

	idx, ok := i.BinSearch(field.String)

	if field.Type == FTExactString {
		if ok {
			buffer.Add(i.capsules[idx].data)
		}
		return
	}

	for idx < len(i.capsules) && strings.HasPrefix(i.capsules[idx].value, field.String) {
		buffer.Add(i.capsules[idx].data)
		idx++
	}
}

func (i *StringIndexCategory) Remove(field Field) {
	if field.Type != FTString {
		return
	}

	if i.Synchronized {
		i.capsulesMutex.Lock()
		defer i.capsulesMutex.Unlock()
	}

	idx, _ := i.BinSearch(field.String)

	for idx < len(i.capsules) && i.capsules[idx].value == field.String {
		if i.capsules[idx].data == field.Value {
			i.capsules = append(i.capsules[:idx], i.capsules[idx+1:]...)
			return
		}
		idx++
	}
}

func (i *StringIndexCategory) BinSearch(value string) (int, bool) {
	if len(i.capsules) == 0 {
		return 0, false
	}

	low, high := 0, len(i.capsules)
	for low < high {
		mid := (low + high) / 2
		midValue := i.capsules[mid].value
		if strings.Compare(midValue, value) < 0 {
			low = mid + 1
		} else {
			high = mid
		}
	}

	return low, low < len(i.capsules) && i.capsules[low].value == value
}

type ResultBuffer interface {
	Add(interface{})
}
