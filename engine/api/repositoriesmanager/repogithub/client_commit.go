package repogithub

import (
	"fmt"
	"reflect"
)

func interfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("interfaceSlice() given a non-slice type")
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}

func arrayContains(array interface{}, s interface{}) bool {
	b := interfaceSlice(array)
	for _, i := range b {
		if reflect.DeepEqual(i, s) {
			return true
		}
	}
	return false
}

func findAncestors(allCommits []Commit, since string) []string {
	ancestors := []string{}
	var i int
	var limit = len(allCommits) * len(allCommits)

ancestorLoop:
	if i > limit {
		return ancestors
	}

	for _, c := range allCommits {
		i++
		if c.Sha == since {
			for _, p := range c.Parents {
				if !arrayContains(ancestors, p.Sha) {
					ancestors = append(ancestors, p.Sha)
					goto ancestorLoop
				}
			}
		} else {
			if arrayContains(ancestors, c.Sha) {
				for _, p := range c.Parents {
					if !arrayContains(ancestors, p.Sha) {
						ancestors = append(ancestors, p.Sha)
						goto ancestorLoop
					}
				}
			}
		}
	}

	fmt.Printf("%s has %d ancestors among %d commits \n", since, len(ancestors), len(allCommits))

	return ancestors
}

func filterCommits(allCommits []Commit, since, until string) []Commit {
	commits := []Commit{}

	sinceAncestors := findAncestors(allCommits, since)
	untilAncestors := findAncestors(allCommits, until)

	//We have to delete all common ancestors between sinceAncestors and untilAncestors
	toDelete := []string{}
	for _, c := range untilAncestors {
		if c == since {
			toDelete = append(toDelete, c)
		}
		if arrayContains(sinceAncestors, c) {
			toDelete = append(toDelete, c)
		}
	}

	for _, d := range toDelete {
		for i, x := range untilAncestors {
			if x == d {
				untilAncestors = append(untilAncestors[:i], untilAncestors[i+1:]...)
			}
		}
	}

	untilAncestors = append(untilAncestors, until)
	for _, c := range allCommits {
		if arrayContains(untilAncestors, c.Sha) {
			commits = append(commits, c)
		}
	}

	return commits
}
