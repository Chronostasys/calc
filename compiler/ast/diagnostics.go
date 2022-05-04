package ast

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

var diagnostics = map[string][]protocol.Diagnostic{}

var diagMu = &sync.RWMutex{}

func ResetErr() {
	errn = 0
	diagMu.Lock()
	diagnostics = make(map[string][]protocol.Diagnostic)
	diagMu.Unlock()
}

func CheckErr() {
	if errn > 0 {
		diagMu.RLock()
		for _, v := range diagnostics {
			for _, v := range v {
				fmt.Println("\033[31m[error]\033[0m:", v.Message)
			}
		}
		diagMu.RUnlock()
		log.Fatalf("compile failed with %d errors.", errn)
	}
}

func GetDiagnostics(file string) []protocol.Diagnostic {
	diagMu.RLock()
	defer diagMu.RUnlock()
	file, _ = filepath.Abs(file)
	if diagnostics[file] == nil {
		return []protocol.Diagnostic{}
	}
	return diagnostics[file]
}

func addDiagnostic(file, msg string, ran protocol.Range, level protocol.DiagnosticSeverity) {

	diagMu.Lock()
	diagnostics[file] = append(diagnostics[file], protocol.Diagnostic{
		Range:    ran,
		Severity: &level,
		Source:   &DIAGNOSTIC_SOURCE,
		Message:  msg,
	})
	diagMu.Unlock()
}
