package service

import (
	"path"
	"sync"
)

var (
	JournalMethodsMatcher MethodMatcher = &cacheableMethodsMatcher{}
)

type MethodMatcher interface {
	Match(method string) bool
}

type cacheableMethodsMatcher struct {
	cache map[string]bool
	lock  sync.RWMutex
	rm    runtimeMatcher
}

func (mm *cacheableMethodsMatcher) Match(method string) bool {
	if len(mm.rm.patterns) == 0 {
		return false
	}

	mm.lock.RLock()
	matched, ok := mm.cache[method]
	mm.lock.RUnlock()
	if ok {
		return matched
	}

	mm.lock.Lock()
	defer mm.lock.Unlock()
	if matched, ok := mm.cache[method]; ok {
		return matched
	}
	matched = mm.rm.Match(method)
	mm.cache[method] = matched
	return matched
}

func NewCacheableMethodMatcher(patterns []string) MethodMatcher {
	return &cacheableMethodsMatcher{
		cache: make(map[string]bool, 0),
		rm:    runtimeMatcher{patterns: patterns},
	}
}

type runtimeMatcher struct {
	patterns []string
}

func (mm runtimeMatcher) Match(method string) bool {
	matched := false
	for _, pattern := range mm.patterns {
		matched, _ = path.Match(pattern, method)
		if matched {
			break
		}
	}
	return matched
}

func NewRuntimeMethodMatcher(patterns []string) MethodMatcher {
	return &runtimeMatcher{
		patterns: patterns,
	}
}
