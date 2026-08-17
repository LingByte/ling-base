package zookeeper_test

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/go-zookeeper/zk"

	lockzk "github.com/LingByte/ling-base/common/lock/zookeeper"
)

type znode struct {
	data     []byte
	children map[string]*znode
}

type fakeConn struct {
	mu          sync.Mutex
	root        *znode
	seq         int64
	createErr   map[string]error
	existsErr   map[string]error
	deleteErr   map[string]error
	childrenErr map[string]error
	getWErr     map[string]error
	watches     map[string][]chan zk.Event
	hideExists  map[string]bool
}

func newFakeConn() *fakeConn {
	return &fakeConn{
		root:        &znode{children: make(map[string]*znode)},
		createErr:   make(map[string]error),
		existsErr:   make(map[string]error),
		deleteErr:   make(map[string]error),
		childrenErr: make(map[string]error),
		getWErr:     make(map[string]error),
		watches:     make(map[string][]chan zk.Event),
		hideExists:  make(map[string]bool),
	}
}

func cleanPath(p string) string {
	p = path.Clean(p)
	if p == "." {
		return "/"
	}
	return p
}

func (f *fakeConn) walk(p string) (*znode, bool) {
	p = cleanPath(p)
	if p == "/" {
		return f.root, true
	}
	cur := f.root
	for _, part := range strings.Split(strings.Trim(p, "/"), "/") {
		if cur.children == nil {
			return nil, false
		}
		next, ok := cur.children[part]
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

func (f *fakeConn) parentAndBase(p string) (*znode, string, error) {
	p = cleanPath(p)
	dir, base := path.Split(p)
	if dir == "" {
		dir = "/"
	}
	parent, ok := f.walk(dir)
	if !ok {
		return nil, "", zk.ErrNoNode
	}
	if parent.children == nil {
		parent.children = make(map[string]*znode)
	}
	return parent, base, nil
}

func (f *fakeConn) Create(p string, data []byte, flags int32, _ []zk.ACL) (string, error) {
	if e := f.createErr[p]; e != nil {
		return "", e
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	p = cleanPath(p)
	parent, base, err := f.parentAndBase(p)
	if err != nil {
		return "", err
	}

	name := base
	if flags&zk.FlagSequence != 0 {
		f.seq++
		name = fmt.Sprintf("%s%010d", base, f.seq)
	} else if _, exists := parent.children[base]; exists {
		return "", zk.ErrNodeExists
	}

	parent.children[name] = &znode{
		data:     append([]byte(nil), data...),
		children: make(map[string]*znode),
	}
	return path.Join(path.Dir(p), name), nil
}

func (f *fakeConn) Delete(p string, _ int32) error {
	if e := f.deleteErr[p]; e != nil {
		return e
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	p = cleanPath(p)
	parent, base, err := f.parentAndBase(p)
	if err != nil {
		return err
	}
	if _, ok := parent.children[base]; !ok {
		return zk.ErrNoNode
	}
	delete(parent.children, base)
	f.notifyWatch(p)
	return nil
}

func (f *fakeConn) Exists(p string) (bool, *zk.Stat, error) {
	if e := f.existsErr[p]; e != nil {
		return false, nil, e
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	p = cleanPath(p)
	if f.hideExists[p] {
		return false, &zk.Stat{}, nil
	}
	_, ok := f.walk(p)
	return ok, &zk.Stat{}, nil
}

func (f *fakeConn) Children(p string) ([]string, *zk.Stat, error) {
	if e := f.childrenErr[p]; e != nil {
		return nil, nil, e
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.walk(cleanPath(p))
	if !ok {
		return nil, nil, zk.ErrNoNode
	}
	out := make([]string, 0, len(n.children))
	for name := range n.children {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, &zk.Stat{}, nil
}

func (f *fakeConn) GetW(p string) ([]byte, *zk.Stat, <-chan zk.Event, error) {
	if e := f.getWErr[p]; e != nil {
		return nil, nil, nil, e
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.walk(cleanPath(p))
	if !ok {
		return nil, nil, nil, zk.ErrNoNode
	}
	ch := make(chan zk.Event, 1)
	p = cleanPath(p)
	f.watches[p] = append(f.watches[p], ch)
	return append([]byte(nil), n.data...), &zk.Stat{}, ch, nil
}

func (f *fakeConn) notifyWatch(p string) {
	for _, ch := range f.watches[p] {
		select {
		case ch <- zk.Event{Type: zk.EventNodeDeleted, Path: p}:
		default:
		}
	}
	delete(f.watches, p)
}

func (f *fakeConn) triggerWatch(p string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notifyWatch(cleanPath(p))
}

var _ lockzk.Conn = (*fakeConn)(nil)
