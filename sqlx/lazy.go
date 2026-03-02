package sqlx

import (
	"context"
	"fmt"
	"sync"

	"github.com/suruiran/cube/internal"
)

type LazyStmt struct {
	*Stmt

	name   string
	rawsql string
	err    error
	once   sync.Once
}

var (
	allLazyStmtsLock sync.Mutex
	allLazyStmts     []*LazyStmt
)

type LazyStmtGroup struct {
	lock sync.Mutex
	eles []*LazyStmt
}

func NewLazyStmtGroup() *LazyStmtGroup { return &LazyStmtGroup{} }

func (g *LazyStmtGroup) New(rawsql string) *LazyStmt {
	ele := NewLazyStmt(rawsql)
	g.lock.Lock()
	g.eles = append(g.eles, ele)
	g.lock.Unlock()
	return ele
}

func (g *LazyStmtGroup) InitAll(ctx context.Context, mapfn func(*LazyStmt) bool) error {
	return initAllLazy(ctx, &g.lock, g.eles, mapfn)
}

func initAllLazy(ctx context.Context, lock *sync.Mutex, lst []*LazyStmt, mapfn func(*LazyStmt) bool) error {
	lock.Lock()
	defer lock.Unlock()

	for _, ele := range lst {
		if mapfn != nil && !mapfn(ele) {
			continue
		}
		if ele.err != nil {
			ele.once = sync.Once{}
		}
		if err := ele.Init(ctx); err != nil {
			return err
		}
	}
	return nil
}

func InitAllLazyStmts(ctx context.Context, mapfn func(*LazyStmt) bool) error {
	return initAllLazy(ctx, &allLazyStmtsLock, allLazyStmts, mapfn)
}

func NewLazyStmt(rawsql string) *LazyStmt {
	ls := &LazyStmt{
		name:   internal.SourceLine(2),
		rawsql: rawsql,
	}
	allLazyStmtsLock.Lock()
	allLazyStmts = append(allLazyStmts, ls)
	allLazyStmtsLock.Unlock()
	return ls
}

func (s *LazyStmt) Name() string {
	return s.name
}

func (s *LazyStmt) WithName(name string) *LazyStmt {
	s.name = name
	return s
}

func (s *LazyStmt) Init(ctx context.Context) error {
	s.once.Do(func() {
		s.Stmt, s.err = NewStmt(ctx, s.rawsql)
	})
	return s.err
}

func (s *LazyStmt) String() string {
	return s.rawsql
}

func (s *LazyStmt) Must() *Stmt {
	if s.Stmt == nil {
		if s.err != nil {
			panic(s.err)
		}
		panic(fmt.Errorf("sqlx: lazy stmt is not initialized, name: `%s`", s.name))
	}
	return s.Stmt
}
