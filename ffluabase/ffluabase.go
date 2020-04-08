package ffluabase

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"github.com/yuin/gopher-lua"
	"strings"
)

func Loader(L *lua.LState) int {
	//注册document类型
	// register functions to the table
	mod := L.SetFuncs(L.NewTable(), exports)
	// register other stuff
	L.SetField(mod, "name", lua.LString("value"))
	// returns the module
	L.Push(mod)
	return 1
}

//返回两个值 一个doc对象 ， 一个err
func newDocument(L *lua.LState) int {
	html := L.ToString(1)
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader([]byte(html)))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LBool(false))
		return 2
	}
	ud := L.NewUserData()
	ud.Value = doc
	L.Push(ud)
	L.Push(lua.LBool(true))
	return 2
}

//查找到element
func find(L *lua.LState) int {
	useData := L.ToUserData(1)
	selector := L.ToString(2)
	//fmt.Println("type:", reflect.TypeOf(useData.Value))
	switch useData.Value.(type) {
	case *goquery.Document:
		doc := useData.Value.(*goquery.Document)
		element := doc.Find(selector)
		if element == nil {
			L.Push(lua.LNil)
			L.Push(lua.LBool(false))
			return 2
		}
		ud := L.NewUserData()
		ud.Value = element
		L.Push(ud)
		L.Push(lua.LBool(true))
	case *goquery.Selection:
		selection := useData.Value.(*goquery.Selection)
		element := selection.Find(selector)
		if element == nil {
			L.Push(lua.LNil)
			L.Push(lua.LBool(false))
			return 2
		}
		ud := L.NewUserData()
		ud.Value = element
		L.Push(ud)
		L.Push(lua.LBool(true))
	default:
		L.Push(lua.LNil)
		L.Push(lua.LBool(false))
		return 2
	}
	return 2
}

func each(L *lua.LState) int {
	userData := L.ToUserData(1)
	selection := userData.Value.(*goquery.Selection)
	var els []*goquery.Selection
	ut := L.NewTable()
	selection.Each(func(i int, selection *goquery.Selection) {
		if selection != nil {
			ud := L.NewUserData()
			ud.Value = selection
			els = append(els, selection)
			ut.Insert(i, ud)
			//log.WithFields(log.Fields{"funcName": "each", "i": i}).Info()
		}
	})

	if ut.Len() == 0 {
		L.Push(lua.LNil)
		L.Push(lua.LBool(false))
		return 2
	}
	L.Push(ut)
	L.Push(lua.LBool(true))
	return 2
}

//获取文本内容
func text(L *lua.LState) int {
	useData := L.ToUserData(1)
	selection := useData.Value.(*goquery.Selection)
	text := selection.Text()
	text = strings.Trim(text, " ")
	text = strings.Trim(text, "\t")
	text = strings.Trim(text, "\n")
	L.Push(lua.LString(text))
	return 1
}

var exports = map[string]lua.LGFunction{
	"newDocument": newDocument,
	"find":        find,
	"text":        text,
	"each":        each,
}
