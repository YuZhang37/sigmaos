package fences

import (
	"sync"

	np "ulambda/ninep"
)

// Keep track of registered fences for a pathname

// A pathname can be protected by several fences
type Entry struct {
	fences map[np.Tfenceid]np.Tfence
}

func makeEntry() *Entry {
	e := &Entry{}
	e.fences = make(map[np.Tfenceid]np.Tfence)
	return e
}

func (e *Entry) getFences() []np.Tfence {
	fences := make([]np.Tfence, 0, len(e.fences))
	for _, f := range e.fences {
		fences = append(fences, f)
	}
	return fences
}

func (e *Entry) insert(f np.Tfence) {
	e.fences[f.FenceId] = f
}

func (e *Entry) del(idf np.Tfenceid) (int, *np.Err) {
	if _, ok := e.fences[idf]; ok {
		delete(e.fences, idf)
		return len(e.fences), nil
	}
	return 0, np.MkErr(np.TErrUnknownFence, idf)
}

func (e *Entry) present(idf np.Tfenceid) bool {
	_, ok := e.fences[idf]
	return ok
}

// For each fenced pathname keep track of the fences for it
type FenceTable struct {
	sync.Mutex
	fencedDirs map[string]*Entry
}

func MakeFenceTable() *FenceTable {
	ft := &FenceTable{}
	ft.fencedDirs = make(map[string]*Entry)
	return ft
}

func (ft *FenceTable) Fences(path np.Path) []np.Tfence {
	ft.Lock()
	defer ft.Unlock()

	fences := make([]np.Tfence, 0)
	for pn, e := range ft.fencedDirs {
		if path.IsParent(np.Split(pn)) {
			fences = append(fences, e.getFences()...)
		}
	}
	return fences
}

func (ft *FenceTable) Insert(path np.Path, f np.Tfence) {
	ft.Lock()
	defer ft.Unlock()

	pn := path.Join()
	e, ok := ft.fencedDirs[pn]
	if !ok {
		e = makeEntry()
		ft.fencedDirs[pn] = e
	}
	e.insert(f)
}

func (ft *FenceTable) Del(path np.Path, idf np.Tfenceid) *np.Err {
	ft.Lock()
	defer ft.Unlock()

	pn := path.Join()
	if e, ok := ft.fencedDirs[pn]; ok {
		n, err := e.del(idf)
		if err != nil {
			return err
		}
		if n == 0 {
			delete(ft.fencedDirs, pn)
		}
		return nil
	}
	return np.MkErr(np.TErrUnknownFence, idf)
}

func (ft *FenceTable) Present(path np.Path, idf np.Tfenceid) bool {
	ft.Lock()
	defer ft.Unlock()

	pn := path.Join()
	if e, ok := ft.fencedDirs[pn]; ok {
		return e.present(idf)
	}
	return false
}
