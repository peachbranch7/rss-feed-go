package pkg

import (
 "strings"
)

type FeedChecker struct {
 Source string
}

func NewFeedChecker(source string) *FeedChecker {
 return &FeedChecker{Source: source}
}

func (fc FeedChecker) RemoveWords(target string) string {
 return strings.Join(strings.Split(fc.Source, target), "")
}