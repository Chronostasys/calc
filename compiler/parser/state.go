package parser

import (
	"path/filepath"
	"sync"

	"github.com/Chronostasys/calc/compiler/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type file struct {
	content    string
	changed    bool
	parsedNode *ast.ProgramNode
}

var activeFiles = map[string]*file{}
var afmu = &sync.RWMutex{}

func SetActiveFile(path, content string) {
	path, _ = filepath.Abs(path)
	afmu.Lock()
	defer afmu.Unlock()
	activeFiles[path] = &file{content: content, changed: true}
}
func SetActiveFileParsed(path, content string, node *ast.ProgramNode) {
	path, _ = filepath.Abs(path)
	afmu.Lock()
	defer afmu.Unlock()

	activeFiles[path] = &file{content: content, changed: false, parsedNode: node}
}

func GetActiveFile(path string) (f *file, ok bool) {
	path, _ = filepath.Abs(path)
	afmu.RLock()
	defer afmu.RUnlock()
	f, ok = activeFiles[path]

	return
}
func ChangeActiveFile(path string, changes []interface{}) {
	path, _ = filepath.Abs(path)
	afmu.Lock()
	defer afmu.Unlock()
	f := activeFiles[path]
	content := f.content
	for _, change := range changes {
		if change_, ok := change.(protocol.TextDocumentContentChangeEvent); ok {
			startIndex, endIndex := change_.Range.IndexesIn(content)
			content = content[:startIndex] + change_.Text + content[endIndex:]
			//log.Debugf("content:\n%s", content)
		} else if change_, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
			content = change_.Text
		}
	}
	activeFiles[path] = &file{content: content, changed: true}
}
