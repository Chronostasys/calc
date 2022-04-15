package main

import (
	"net/url"
	"path"
	"runtime"
	"strings"

	"github.com/Chronostasys/calc/compiler/ast"
	"github.com/Chronostasys/calc/compiler/parser"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

const lsName = "calcls"

var version string = "0.0.1"
var handler protocol.Handler
var root = ""

func main() {

	handler = protocol.Handler{
		Initialize:  initialize,
		Initialized: initialized,
		Shutdown:    shutdown,
		SetTrace:    setTrace,
		TextDocumentDidChange: func(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
			// params.TextDocument.URI
			url, _ := url.ParseRequestURI(params.TextDocument.URI)
			p := url.Path
			if runtime.GOOS == "windows" {
				p = p[1:]
			}
			parser.ChangeActiveFile(p, params.ContentChanges)
			parser.GetDiagnostics(path.Dir(p))

			// protocol.Diagnostic
			// fmt.Println(params.TextDocument.URI)
			context.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
				URI:         params.TextDocument.URI,
				Diagnostics: ast.GetDiagnostics(p),
			})
			return nil
		},
		TextDocumentDidOpen: func(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
			url, _ := url.ParseRequestURI(params.TextDocument.URI)
			p := url.Path
			if runtime.GOOS == "windows" {
				p = strings.Trim(p, "/")
			}
			parser.SetActiveFile(p, params.TextDocument.Text)
			parser.GetDiagnostics(path.Dir(p))

			// protocol.Diagnostic
			// fmt.Println(params.TextDocument.URI)
			context.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
				URI:         params.TextDocument.URI,
				Diagnostics: ast.GetDiagnostics(p),
			})
			return nil
		},
		TextDocumentCompletion: func(context *glsp.Context, params *protocol.CompletionParams) (interface{}, error) {
			url, _ := url.ParseRequestURI(params.TextDocument.URI)
			p := url.Path
			if runtime.GOOS == "windows" {
				p = strings.Trim(p, "/")
			}

			sp := p

			var ls interface{}
			if params.Context.TriggerCharacter != nil && *params.Context.TriggerCharacter == "." {
				ls = ast.GetDotAutocomplete(sp, params.Position.Line)
			} else {
				ls = ast.GetAutocomplete(sp, params.Position.Line)
			}
			return ls, nil
		},
	}

	server := server.NewServer(&handler, lsName, false)

	server.RunStdio()
}

func initialize(context *glsp.Context, params *protocol.InitializeParams) (interface{}, error) {
	root = *params.RootPath
	capabilities := handler.CreateServerCapabilities()
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{"."},
	}
	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &version,
		},
	}, nil
}

func initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	return nil
}

func shutdown(context *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}
