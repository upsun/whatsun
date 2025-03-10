package celfuncs

type FuncDoc struct {
	Comment     string
	Description string
	Args        []ArgDoc
}

type ArgDoc struct {
	Name    string
	Comment string
}

type Docs struct {
	funcs map[string]*FuncDoc
}

func (d *Docs) GetFunction(name string) (*FuncDoc, bool) {
	fd, ok := d.funcs[name]
	return fd, ok
}

func (d *Docs) AddFunction(name string, fd FuncDoc) {
	if d == nil {
		return
	}
	if d.funcs == nil {
		d.funcs = make(map[string]*FuncDoc)
	}
	d.funcs[name] = &fd
}
