package ast

import (
	"path/filepath"
	"sort"
	"sync"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type refPos struct {
	pos        protocol.Location
	start, end uint32
}

var refMu = &sync.RWMutex{}

var refMap = map[string]map[uint32][]refPos{}

func addRef(f string, ran protocol.Range, pos protocol.Location) {
	f, _ = filepath.Abs(f)
	if pos.Range.Start.Line == 0 && pos.Range.Start.Character == 0 {
		return
	}
	refMu.Lock()
	defer refMu.Unlock()
	if refMap[f] == nil {
		refMap[f] = map[uint32][]refPos{}
	}
	frefMap := refMap[f]
	ln := ran.Start.Line
	frefMap[ln] = append(frefMap[ln], refPos{pos: pos, start: ran.Start.Character, end: ran.End.Character})
	sort.Slice(frefMap[ln], func(i, j int) bool {
		return frefMap[ln][i].start < frefMap[ln][j].start
	})
}

func GetRefPos(f string, pos protocol.Position) []protocol.Location {
	f, _ = filepath.Abs(f)
	refMu.RLock()
	defer refMu.RUnlock()
	if refMap[f] == nil {
		return []protocol.Location{}
	}
	frefMap := refMap[f]
	ls := frefMap[pos.Line]
	if ls == nil {
		return []protocol.Location{}
	}
	for _, v := range ls {
		if pos.Character >= v.start && pos.Character <= v.end {
			return []protocol.Location{v.pos}
		}
	}
	return []protocol.Location{}
}
