package hx

import (
	"reflect"
	"regexp"
	"runtime"
)

var funRegexp = regexp.MustCompile(`(?:^|/)[^/.]+\.([A-Za-z_][A-Za-z0-9_]*)$`)

func funName(fn HandlerFunc) (name string, ok bool) {
	if fn == nil {
		return "", false
	}
	rf := runtime.FuncForPC(reflect.ValueOf(fn).Pointer())
	if rf == nil {
		return "", false
	}
	name = rf.Name()
	if !funRegexp.MatchString(name) {
		return "", false
	}
	return name, true
}
