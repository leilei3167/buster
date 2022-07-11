package dir

import "buster/lib"

type OptionsDir struct {
	lib.HTTPOptions
	Extensions                 string
	ExtensionsParsed           lib.StringSet
	StatusCodes                string
	StatusCodesParsed          lib.IntSet
	StatusCodesBlacklist       string
	StatusCodesBlacklistParsed lib.IntSet
	UseSlash                   bool
	HideLength                 bool
	Expanded                   bool
	NoStatus                   bool
	DiscoverBackup             bool
	ExcludeLength              []int
}

func NewOptionsDir() *OptionsDir {
	return &OptionsDir{
		StatusCodesParsed:          lib.NewIntSet(),
		StatusCodesBlacklistParsed: lib.NewIntSet(),
		ExtensionsParsed:           lib.NewStringSet(),
	}
}
