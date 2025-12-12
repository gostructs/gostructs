package main

// VarDecl - used in variable declaration
type VarDecl struct{ V int }

// FuncParam - used as function parameter
type FuncParam struct{ P int }

// FuncReturn - used as function return type
type FuncReturn struct{ R int }

// FieldType - used as field in another struct
type FieldType struct{ F int }

// Container - uses FieldType
type Container struct {
	Field FieldType
}

// CompositeLit - used in composite literal
type CompositeLit struct{ C int }

// TypeAssert - used in type assertion
type TypeAssert struct{ T int }

// EmbeddedUsed - embedded in UsedEmbedder
type EmbeddedUsed struct{ E int }

// UsedEmbedder - embeds EmbeddedUsed and is used
type UsedEmbedder struct {
	EmbeddedUsed
}

// PointerType - used as pointer
type PointerType struct{ Ptr int }

// SliceType - used in slice
type SliceType struct{ S int }

// MapKey - used as map key
type MapKey struct{ K int }

// MapValue - used as map value
type MapValue struct{ M int }

// ChanType - used in channel
type ChanType struct{ Ch int }

func takeParam(p FuncParam) {}
func returnVal() FuncReturn { return FuncReturn{} }

func main() {
	var _ VarDecl
	takeParam(FuncParam{})
	_ = returnVal()
	_ = Container{}
	_ = CompositeLit{C: 1}

	var x interface{} = TypeAssert{}
	_, _ = x.(TypeAssert)

	_ = UsedEmbedder{}
	_ = &PointerType{}
	_ = []SliceType{}
	_ = map[MapKey]MapValue{}
	_ = make(chan ChanType)
}
