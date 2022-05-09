package ast

import (
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/Chronostasys/calc/compiler/helper"
	"github.com/llir/llvm/ir"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type autoComplete struct {
	completes [2][]protocol.CompletionItem
	scope     string
	m         map[string]struct{}
	leading   string
	tp        bool
}

var autocompleteMap = map[string]map[uint32]autoComplete{}
var autocompleteMu = &sync.RWMutex{}

func setAutocomplete(file, mod string, line uint32, cmpls []protocol.CompletionItem) {
	autocompleteMu.Lock()
	defer autocompleteMu.Unlock()
	file, _ = filepath.Abs(file)
	if autocompleteMap[file] == nil {
		autocompleteMap[file] = map[uint32]autoComplete{}
	}
	autocompleteMap[file][line] = autoComplete{
		completes: [2][]protocol.CompletionItem{cmpls, autocompleteMap[file][line].completes[1]},
		scope:     mod,
	}
}
func setDotAutocomplete(file, mod string, line uint32, cmpls []protocol.CompletionItem) {
	autocompleteMu.Lock()
	defer autocompleteMu.Unlock()
	file, _ = filepath.Abs(file)
	if autocompleteMap[file] == nil {
		autocompleteMap[file] = map[uint32]autoComplete{}
	}
	autocompleteMap[file][line] = autoComplete{
		completes: [2][]protocol.CompletionItem{autocompleteMap[file][line].completes[0], cmpls},
		scope:     mod,
	}
}

var mu = &sync.RWMutex{}

func GetAutocomplete(file string, line uint32) []protocol.CompletionItem {
	autocompleteMu.RLock()
	file, _ = filepath.Abs(file)
	ls := autocompleteMap[file][line].completes[0]
	ac := autocompleteMap[file][line]
	autocompleteMu.RUnlock()
	ScopeMapMu.RLock()
	sc := ScopeMap[ac.scope]
	ScopeMapMu.RUnlock()
	if sc == nil {
		return ls
	}
	if ac.tp {
		return ls
	}
	mu.RLock()
	cmpls := getCurrentScopeAutoComplete(ac.m, sc, ac.leading, false, ac.tp)
	mu.RUnlock()
	return append(ls, cmpls...)
}

func GetDotAutocomplete(file string, line uint32) []protocol.CompletionItem {
	autocompleteMu.RLock()
	defer autocompleteMu.RUnlock()
	file, _ = filepath.Abs(file)
	return autocompleteMap[file][line].completes[1]
}

func getCurrentScopeAutoComplete(m map[string]struct{}, sc *Scope, leading string, extern, tponly bool) []protocol.CompletionItem {
	if m == nil {
		m = map[string]struct{}{}
	}
	cmpls := []protocol.CompletionItem{}

	for k, v := range sc.types {
		k = helper.LastBlock(k)
		if extern && !unicode.IsUpper(rune(k[0])) {
			continue
		}
		if strings.Index(k, leading) < 0 {
			continue
		}
		if _, ok := m[k]; ok {
			continue
		}
		m[k] = struct{}{}
		kind := protocol.CompletionItemKindStruct
		if _, ok := v.structType.(*interf); ok {
			kind = protocol.CompletionItemKindInterface
		}
		if strings.Contains(k, "}") || strings.Contains(k, ">") { // generic和匿名结构体
			continue
		}
		ins := k
		if !tponly {
			ins = ins + "{}"
		}
		cmpls = append(cmpls, protocol.CompletionItem{
			Label:      k,
			Kind:       &kind,
			InsertText: &ins,
		})
	}
	for k := range sc.genericStructs {
		k = helper.LastBlock(k)
		if extern && !unicode.IsUpper(rune(k[0])) {
			continue
		}
		if strings.Index(k, leading) < 0 {
			continue
		}
		if _, ok := m[k]; ok {
			continue
		}
		m[k] = struct{}{}
		kind := protocol.CompletionItemKindStruct
		ins := k
		if !tponly {
			ins = ins + "{}"
		}
		cmpls = append(cmpls, protocol.CompletionItem{
			Label:      k,
			Kind:       &kind,
			InsertText: &ins,
		})
	}
	if tponly {
		return cmpls
	}
	for k, v := range sc.vartable {
		if v.attachedFunc {
			continue
		}
		if externMap[k] {
			continue
		}
		k = helper.LastBlock(k)
		if extern && !unicode.IsUpper(rune(k[0])) {
			continue
		}
		if strings.Index(k, leading) < 0 {
			continue
		}
		if strings.Contains(k, "<") {
			continue
		}
		if _, ok := m[k]; ok {
			continue
		}
		m[k] = struct{}{}
		kind := protocol.CompletionItemKindVariable
		ins := k
		if _, ok := v.v.(*ir.Func); ok {
			kind = protocol.CompletionItemKindFunction
			ins = ins + "()"
		}
		cmpls = append(cmpls, protocol.CompletionItem{
			Label:      k,
			Kind:       &kind,
			InsertText: &ins,
		})
	}
	for k := range sc.genericFuncs {
		if genericAttached[k] {
			continue
		}
		if externMap[k] {
			continue
		}
		k = helper.LastBlock(k)
		if extern && !unicode.IsUpper(rune(k[0])) {
			continue
		}
		if strings.Index(k, leading) < 0 {
			continue
		}
		if _, ok := m[k]; ok {
			continue
		}
		m[k] = struct{}{}
		kind := protocol.CompletionItemKindFunction
		ins := k + "()"
		cmpls = append(cmpls, protocol.CompletionItem{
			Label:      k,
			Kind:       &kind,
			InsertText: &ins,
		})
	}
	return cmpls
}

func genAutoComplete(file string, line uint32, sc *Scope, leading string, set, extern, tp bool) []protocol.CompletionItem {
	m := map[string]struct{}{}
	cmpls := []protocol.CompletionItem{}
	orisc := sc
	for {
		cmpls = append(cmpls, getCurrentScopeAutoComplete(m, sc, leading, extern, tp)...)
		sc = sc.parent
		if sc == nil {
			break
		}
	}
	if !set {
		return cmpls
	}
	autocompleteMu.Lock()
	defer autocompleteMu.Unlock()
	if autocompleteMap[file] == nil {
		autocompleteMap[file] = map[uint32]autoComplete{}
	}
	autocompleteMap[file][line] = autoComplete{
		completes: [2][]protocol.CompletionItem{cmpls, autocompleteMap[file][line].completes[1]},
		m:         m,
		scope:     orisc.Pkgname,
		leading:   leading,
		tp:        tp,
	}
	return cmpls
}
