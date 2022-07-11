package helper

import (
	"buster/lib"
	"fmt"
	"strconv"
	"strings"
)

func ParseExtentions(ext string) (lib.StringSet, error) {
	if ext == "" {
		return lib.StringSet{}, nil
	}
	ret := lib.NewStringSet()

	for _, e := range strings.Split(ext, ",") {
		e = strings.TrimSpace(e)
		ret.Add(strings.TrimPrefix(e, ".")) //将去除前缀的拓展名作为key加入
	}
	return ret, nil
}

func ParseCommaSeparatedInt(inputString string) (lib.IntSet, error) {
	if inputString == "" {
		return lib.IntSet{}, nil
	}

	ret := lib.NewIntSet()
	for _, c := range strings.Split(inputString, ",") {
		c = strings.TrimSpace(c)
		i, err := strconv.Atoi(c)
		if err != nil {
			return lib.IntSet{}, fmt.Errorf("invalid string given: %s", c)
		}
		ret.Add(i)
	}
	return ret, nil
}
