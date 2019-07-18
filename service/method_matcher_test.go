package service

import "testing"

var (
	patterns = []string{
		"mdm-master/group/method",
		"mdm-master/group2/method",
		"mdm-master/group3/*",
	}
	cases = []struct {
		Method string
		Result bool
	}{
		{Method: "mdm-master/group/method", Result: true},
		{Method: "mdm-master/group/method2", Result: false},
		{Method: "mdm-master/group2/method3", Result: false},
		{Method: "mdm-master/group2/method", Result: true},
		{Method: "mdm-master/group3/some", Result: true},
		{Method: "mdm-master/group3/", Result: true},
	}
)

func TestCacheableMethodsMatcher_Match(t *testing.T) {
	matcher := NewCacheableMethodMatcher(patterns)
	for _, c := range cases {
		res := matcher.Match(c.Method)
		if c.Result != res {
			t.Error(c)
		}
	}
}

func BenchmarkCacheableMethodsMatcher_Match(b *testing.B) {
	matcher := NewCacheableMethodMatcher(patterns)
	for i := 0; i < b.N; i++ {
		for _, c := range cases {
			_ = matcher.Match(c.Method)
		}
	}
}

func BenchmarkRuntimeMatcher_Match(b *testing.B) {
	matcher := NewRuntimeMethodMatcher(patterns)
	for i := 0; i < b.N; i++ {
		for _, c := range cases {
			_ = matcher.Match(c.Method)
		}
	}
}
