package ubuntu_sdk_tools

import (
	"strings"
	"regexp"
)

var _find_unsafe = regexp.MustCompile("[^\\w@%+=:,./-]")
func QuoteString (s string) string{
	//Return a shell-escaped version of the string *s*.
	if len(s) == 0 {
		return "''"
	}

	if !_find_unsafe.MatchString(s) {
		return s
	}


	// use single quotes, and put single quotes into double quotes
	// the string $'b is then quoted as '$'"'"'b'

	return "'" + strings.Replace(s, "'", "'\"'\"'", -1) + "'"
}
