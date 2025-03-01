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

var FuncDocs = map[string]FuncDoc{}
