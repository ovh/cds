package gluare

import (
	"github.com/yuin/gopher-lua"
	"regexp"
	"strings"
)

func Loader(L *lua.LState) int {
	mod := L.SetFuncs(L.NewTable(), exports)
	mod.RawSetString("gmatch", L.NewClosure(reGmatch, L.NewFunction(reGmatchIter)))
	L.Push(mod)
	return 1
}

var exports = map[string]lua.LGFunction{
	"find":  reFind,
	"gsub":  reGsub,
	"match": reMatch,
	"quote": reQuote,
}

func reQuote(L *lua.LState) int {
	str := L.CheckString(1)
	L.Push(lua.LString(regexp.QuoteMeta(str)))
	return 1
}

func reFind(L *lua.LState) int {
	str := L.CheckString(1)
	pattern := L.CheckString(2)
	if len(str) == 0 && len(pattern) == 0 {
		L.Push(lua.LNumber(1))
		L.Push(lua.LNumber(0))
		return 2
	}
	init := luaIndex2StringIndex(str, L.OptInt(3, 1), true)
	plain := false
	if L.GetTop() == 4 {
		plain = lua.LVAsBool(L.Get(4))
	}

	if plain {
		pos := strings.Index(str[init:], pattern)
		if pos < 0 {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(lua.LNumber(init+pos) + 1)
		L.Push(lua.LNumber(init + pos + len(pattern)))
		return 2
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		L.RaiseError(err.Error())
	}
	stroffset := str[init:]
	positions := re.FindStringSubmatchIndex(stroffset)
	if positions == nil || (len(positions) > 1 && positions[2] < 0) {
		L.Push(lua.LNil)
		return 1
	}
	npos := len(positions)
	L.Push(lua.LNumber(init+positions[0]) + 1)
	L.Push(lua.LNumber(init + positions[npos-1]))
	for i := 2; i < npos; i += 2 {
		L.Push(lua.LString(stroffset[positions[i]:positions[i+1]]))
	}
	return npos/2 + 1
}

func reGsub(L *lua.LState) int {
	str := L.CheckString(1)
	pat := L.CheckString(2)
	L.CheckTypes(3, lua.LTString, lua.LTTable, lua.LTFunction)
	repl := L.CheckAny(3)
	limit := L.OptInt(4, -1)

	re, err := regexp.Compile(pat)
	if err != nil {
		L.RaiseError(err.Error())
	}
	matches := re.FindAllStringSubmatchIndex(str, limit)
	if matches == nil || len(matches) == 0 {
		L.SetTop(1)
		L.Push(lua.LNumber(0))
		return 2
	}
	switch lv := repl.(type) {
	case lua.LString:
		L.Push(lua.LString(reGsubStr(str, re, string(lv), matches)))
	case *lua.LTable:
		L.Push(lua.LString(reGsubTable(L, str, lv, matches)))
	case *lua.LFunction:
		L.Push(lua.LString(reGsubFunc(L, str, lv, matches)))
	}
	L.Push(lua.LNumber(len(matches)))
	return 2
}

type replaceInfo struct {
	Indicies []int
	String   string
}

func reGsubDoReplace(str string, info []replaceInfo) string {
	offset := 0
	buf := []byte(str)
	for _, replace := range info {
		oldlen := len(buf)
		b1 := append([]byte(""), buf[0:offset+replace.Indicies[0]]...)
		b2 := []byte("")
		index2 := offset + replace.Indicies[1]
		if index2 <= len(buf) {
			b2 = append(b2, buf[index2:len(buf)]...)
		}
		buf = append(b1, replace.String...)
		buf = append(buf, b2...)
		offset += len(buf) - oldlen
	}
	return string(buf)
}

func reGsubStr(str string, re *regexp.Regexp, repl string, matches [][]int) string {
	infoList := make([]replaceInfo, 0, len(matches))
	for _, match := range matches {
		start, end := match[0], match[1]
		if end < 0 {
			continue
		}
		buf := make([]byte, 0, end-start)
		buf = re.ExpandString(buf, repl, str, match)
		infoList = append(infoList, replaceInfo{[]int{start, end}, string(buf)})
	}

	return reGsubDoReplace(str, infoList)
}

func reGsubTable(L *lua.LState, str string, repl *lua.LTable, matches [][]int) string {
	infoList := make([]replaceInfo, 0, len(matches))
	for _, match := range matches {
		var key string
		start, end := match[0], match[1]
		if end < 0 {
			continue
		}
		if len(match) > 2 { // has captures
			key = str[match[2]:match[3]]
		} else {
			key = str[match[0]:match[1]]
		}
		value := L.GetField(repl, key)
		if !lua.LVIsFalse(value) {
			infoList = append(infoList, replaceInfo{[]int{start, end}, lua.LVAsString(value)})
		}
	}
	return reGsubDoReplace(str, infoList)
}

func reGsubFunc(L *lua.LState, str string, repl *lua.LFunction, matches [][]int) string {
	infoList := make([]replaceInfo, 0, len(matches))
	for _, match := range matches {
		start, end := match[0], match[1]
		if end < 0 {
			continue
		}
		L.Push(repl)
		nargs := 0
		if len(match) > 2 { // has captures
			for i := 2; i < len(match); i += 2 {
				L.Push(lua.LString(str[match[i]:match[i+1]]))
				nargs++
			}
		} else {
			L.Push(lua.LString(str[start:end]))
			nargs++
		}
		L.Call(nargs, 1)
		ret := L.Get(-1)
		L.Pop(1)
		if !lua.LVIsFalse(ret) {
			infoList = append(infoList, replaceInfo{[]int{start, end}, lua.LVAsString(ret)})
		}
	}
	return reGsubDoReplace(str, infoList)
}

type reMatchData struct {
	str     string
	pos     int
	matches [][]int
}

func reGmatchIter(L *lua.LState) int {
	md := L.CheckUserData(1).Value.(*reMatchData)
	str := md.str
	matches := md.matches
	idx := md.pos
	md.pos += 1
	if idx == len(matches) {
		return 0
	}
	L.Push(L.Get(1))
	match := matches[idx]
	if len(match) == 2 {
		L.Push(lua.LString(str[match[0]:match[1]]))
		return 1
	}

	for i := 2; i < len(match); i += 2 {
		L.Push(lua.LString(str[match[i]:match[i+1]]))
	}
	return len(match)/2 - 1
}

func reGmatch(L *lua.LState) int {
	str := L.CheckString(1)
	pattern := L.CheckString(2)
	re, err := regexp.Compile(pattern)
	if err != nil {
		L.RaiseError(err.Error())
	}
	L.Push(L.Get(lua.UpvalueIndex(1)))
	ud := L.NewUserData()
	ud.Value = &reMatchData{str, 0, re.FindAllStringSubmatchIndex(str, -1)}
	L.Push(ud)
	return 2
}

func reMatch(L *lua.LState) int {
	str := L.CheckString(1)
	pattern := L.CheckString(2)
	offset := L.OptInt(3, 1)
	l := len(str)
	if offset < 0 {
		offset = l + offset + 1
	}
	offset--
	if offset < 0 {
		offset = 0
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		L.RaiseError(err.Error())
	}
	str = str[offset:]
	subs := re.FindStringSubmatchIndex(str)
	nsubs := len(subs) / 2
	switch nsubs {
	case 0:
		L.Push(lua.LNil)
		return 1
	case 1:
		L.Push(lua.LString(str[subs[0]:subs[1]]))
		return 1
	default:
		for i := 2; i < len(subs); i += 2 {
			L.Push(lua.LString(str[subs[i]:subs[i+1]]))
		}
		return nsubs - 1
	}

}

func luaIndex2StringIndex(str string, i int, start bool) int {
	if start && i != 0 {
		i -= 1
	}
	l := len(str)
	if i < 0 {
		i = l + i + 1
	}
	if 0 > i {
		i = 0
	}
	if !start && i > l {
		i = l
	}
	return i
}

//
