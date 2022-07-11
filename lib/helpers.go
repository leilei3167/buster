package lib

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	// VERSION contains the current gobuster version
	VERSION = "3.1.0"
)

// DefaultUserAgent returns the default user agent to use in HTTP requests
func DefaultUserAgent() string {
	return fmt.Sprintf("gobuster/%s", VERSION)
}

type IntSet struct {
	Set map[int]bool
}

type StringSet struct {
	Set map[string]bool
}

func NewStringSet() StringSet {
	return StringSet{Set: make(map[string]bool)}
}

func (set *StringSet) Add(s string) bool {
	_, found := set.Set[s]
	set.Set[s] = true
	return !found
}
func (set *StringSet) AddRange(ss []string) {
	for _, s := range ss {
		set.Set[s] = true
	}
}
func (set *StringSet) Contains(s string) bool {
	_, found := set.Set[s]
	return found
}
func (set *StringSet) ContainsAny(ss []string) bool {
	for _, s := range ss {
		if set.Set[s] {
			return true
		}
	}
	return false
}

func (set *StringSet) Length() int {
	return len(set.Set)
}

// Stringify 将StringSet转换为string输出
func (set *StringSet) Stringify() string {
	values := make([]string, len(set.Set))
	var i int
	//只需要map中的key
	for s := range set.Set {
		values[i] = s
		i++
	}

	return strings.Join(values, ",")

}

func NewIntSet() IntSet {
	return IntSet{Set: make(map[int]bool)}
}

// Add adds an element to a set
func (set *IntSet) Add(i int) bool {
	_, found := set.Set[i]
	set.Set[i] = true
	return !found
}

// Contains tests if an element is in a set
func (set *IntSet) Contains(i int) bool {
	_, found := set.Set[i]
	return found
}

func (set *IntSet) Stringify() string {
	values := make([]int, len(set.Set))
	i := 0
	for s := range set.Set {
		values[i] = s
		i++
	}
	sort.Ints(values)
	delim := ","
	//Fields将字符串根据空格拆分,之后再用,连接为字符串,并去除掉[]元素
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(values)), delim), "[]")
}
func (set *IntSet) Length() int {
	return len(set.Set)
}

func lineConut(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024) //32kb

	count := 1
	lineSep := []byte{'\n'}

	//每次读取32kb,并统计其中出现的换行符
	for {
		c, err := r.Read(buf)
		count = count + bytes.Count(buf[:c], lineSep)

		switch {
		case errors.Is(err, io.EOF):
			return count, nil
		case err != nil:
			return count, err
		}

	}

}
