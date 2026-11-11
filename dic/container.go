package dic

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"slices"
)

var errorInterfaceType = reflect.TypeFor[error]()

type _DiFnc struct {
	deps        []reflect.Type
	fnc         reflect.Value
	lastRvIsErr bool
	done        bool
}

func (ele *_DiFnc) call(args []reflect.Value) (rvs []reflect.Value, ok bool) {
	defer func() {
		rv := recover()
		if rv == nil {
			return
		}
		if IsTokenNotFound(rv) {
			ok = false
			rvs = nil
			return
		}
		panic(rv)
	}()
	return ele.fnc.Call(args), true
}

type Container[T comparable] struct {
	execed       bool
	fncs         []*_DiFnc
	asyncfuncs   []*_DiFnc
	valpool      map[reflect.Type]reflect.Value
	tokenvalpool map[reflect.Type]map[T]reflect.Value
}

func New[T comparable]() *Container[T] {
	return &Container[T]{
		valpool:      make(map[reflect.Type]reflect.Value),
		tokenvalpool: make(map[reflect.Type]map[T]reflect.Value),
	}
}

func (dic *Container[T]) appendone(v reflect.Value) {
	var vtype reflect.Type
	var maytoken sql.Null[T]
	if v.Type().Implements(tokenValueInterfaceType) {
		var anytoken any
		anytoken, vtype, v = (v.Interface().(__I_private_dic_token_value__)).__private_dic_token_value__()
		maytoken = sql.Null[T]{V: anytoken.(T), Valid: true}
	} else {
		vtype = v.Type()
	}

	if maytoken.Valid {
		tvm := dic.tokenvalpool[vtype]
		if tvm == nil {
			tvm = make(map[T]reflect.Value)
		}
		if _, ok := tvm[maytoken.V]; ok {
			panic(fmt.Errorf("cube.dic: type `%v`, token `%v` is already registered", vtype, maytoken.V))
		}
		tvm[maytoken.V] = v
		dic.tokenvalpool[vtype] = tvm
		return
	}

	if _, ok := dic.valpool[vtype]; ok {
		panic(fmt.Errorf("cube.dic: `%s` is already registered", vtype))
	}
	dic.valpool[vtype] = v
}

func (dic *Container[T]) append(v reflect.Value) {
	if v.Kind() == reflect.Slice && v.Type().Elem().Implements(tokenValueInterfaceType) {
		for i := range v.Len() {
			dic.appendone(v.Index(i))
		}
		return
	}
	dic.appendone(v)
}

func (dic *Container[T]) get(k reflect.Type) (reflect.Value, bool) {
	v, ok := dic.valpool[k]
	return v, ok
}

type errTokenNotFound struct {
	rtype reflect.Type
	token any
}

func (e *errTokenNotFound) Error() string {
	return fmt.Sprintf("cube.dic: type `%s` token `%s` not found", e.rtype, e.token)
}

var _ error = (*errTokenNotFound)(nil)

func IsTokenNotFound(val any) bool {
	e, ok := val.(error)
	if !ok {
		return false
	}
	var target *errTokenNotFound
	return errors.As(e, &target)
}

func (dic *Container[T]) getbytoken(token T, k reflect.Type) (reflect.Value, bool) {
	tvm, ok := dic.tokenvalpool[k]
	if !ok {
		return reflect.Value{}, false
	}
	v, ok := tvm[token]
	if !ok {
		panic(&errTokenNotFound{token: token, rtype: k})
	}
	return v, true
}

func (dic *Container[T]) mkfnc(fnc any) *_DiFnc {
	rv := reflect.ValueOf(fnc)
	if !rv.IsValid() || rv.Kind() != reflect.Func {
		panic(fmt.Errorf("cube.dic: `%s` is not a function", fnc))
	}

	ele := &_DiFnc{fnc: rv}
	for in := range rv.Type().Ins() {
		ele.deps = append(ele.deps, in)
	}

	outnum := rv.Type().NumOut()
	if outnum > 1 {
		ele.lastRvIsErr = rv.Type().Out(outnum - 1).Implements(errorInterfaceType)
	}
	return ele
}

var ErrContainerAlreadyExecuted = errors.New("cube.dic: container already executed")

func (dic *Container[T]) Register(fnc any) *Container[T] {
	if dic.execed {
		panic(ErrContainerAlreadyExecuted)
	}
	dic.fncs = append(dic.fncs, dic.mkfnc(fnc))
	return dic
}

func (dic *Container[T]) OnReady(fnc any) *Container[T] {
	if dic.execed {
		panic(ErrContainerAlreadyExecuted)
	}
	ele := dic.mkfnc(fnc)
	dic.asyncfuncs = append(dic.asyncfuncs, ele)
	return dic
}

func (dic *Container[T]) Get[V any](dest *V, token ...T) bool {
	dt := reflect.TypeOf(dest).Elem()
	if len(token) > 0 {
		val, ok := dic.getbytoken(token[0], dt)
		if !ok {
			return false
		}
		reflect.ValueOf(dest).Elem().Set(val)
		return true
	}
	v, ok := dic.get(dt)
	if !ok {
		return false
	}
	reflect.ValueOf(dest).Elem().Set(v)
	return true
}

func (dic *Container[T]) GetByToken[V any](dest *V, token T) *Container[T] {
	dt := reflect.TypeOf(dest).Elem()
	val, ok := dic.getbytoken(token, dt)
	if !ok {
		panic(&errTokenNotFound{token: token, rtype: dt})
	}
	reflect.ValueOf(dest).Elem().Set(val)
	return dic
}

func (dic *Container[T]) execAsync(ele *_DiFnc) bool {
	if ele.done {
		return false
	}
	args := dic.deps4ele(ele)
	if len(args) == len(ele.deps) {
		rvs, ok := ele.call(args)
		if !ok {
			return false
		}
		if ele.lastRvIsErr {
			lastrv := rvs[len(rvs)-1]
			if lastv := lastrv.Interface(); lastv != nil {
				panic(
					fmt.Errorf(
						"cube.dic: async function `%s`, return error: %w",
						runtime.FuncForPC(ele.fnc.Pointer()).Name(), lastv.(error),
					),
				)
			}
		}
		ele.done = true
		return true
	}
	return false
}

func (dic *Container[T]) execAllAsyncEles() {
	for _, ele := range dic.asyncfuncs {
		dic.execAsync(ele)
	}
}

func (dic *Container[T]) deps4ele(ele *_DiFnc) []reflect.Value {
	var args []reflect.Value
	for _, argtype := range ele.deps {
		av, ok := dic.get(argtype)
		if !ok {
			break
		}
		args = append(args, av)
	}
	return args
}

func (dic *Container[T]) Run() {
	if dic.execed {
		panic(ErrContainerAlreadyExecuted)
	}
	dic.execed = true

	for {
		var remains []*_DiFnc
		for _, ele := range dic.fncs {
			if ele.done {
				continue
			}
			remains = append(remains, ele)
		}

		if len(remains) < 1 {
			break
		}

		donecount := 0
		for _, ele := range remains {
			args := dic.deps4ele(ele)
			if len(args) == len(ele.deps) {
				rvs, ok := ele.call(args)
				if !ok {
					continue
				}

				if ele.lastRvIsErr {
					lastrv := rvs[len(rvs)-1]
					if lastv := lastrv.Interface(); lastv != nil {
						panic(fmt.Errorf("cube.dic: function `%s` returned error: %w", runtime.FuncForPC(ele.fnc.Pointer()).Name(), lastv.(error)))
					}
					rvs = rvs[0 : len(rvs)-1]
				}

				for _, rv := range rvs {
					dic.append(rv)
				}
				ele.done = true
				donecount++
			}
		}

		if donecount < 1 {
			fmt.Println("Remaining functions that could not be executed due to missing dependencies:")
			for _, ele := range remains {
				if ele.done {
					continue
				}
				fnName := runtime.FuncForPC(ele.fnc.Pointer()).Name()
				fmt.Printf("  - Function: %s\n", fnName)
			}
			panic(errors.New("cube.dic: cannot resolve dependencies"))
		}
		dic.execAllAsyncEles()
	}

	dic.execAllAsyncEles()

	udi := slices.IndexFunc(dic.asyncfuncs, func(ele *_DiFnc) bool { return !ele.done })
	if udi > -1 {
		fmt.Println("Remaining async-functions that could not be executed due to missing dependencies:")
		for _, ele := range dic.asyncfuncs {
			if ele.done {
				continue
			}
			fnName := runtime.FuncForPC(ele.fnc.Pointer()).Name()
			fmt.Printf("  - Function: %s\n", fnName)
		}
		panic(errors.New("cube.dic: cannot resolve dependencies"))
	}
}
