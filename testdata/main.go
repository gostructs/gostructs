package main

// Used - referenced in main()
type Used struct {
	Name string
}

// Unused - never referenced
type Unused struct {
	Data int
}

// UsedInUnused - only referenced by Unused2
type UsedInUnused struct {
	Value int
}

// Unused2 - references UsedInUnused but itself unused
type Unused2 struct {
	Inner UsedInUnused
}

// WithMethod - has method but never instantiated
type WithMethod struct{}

func (w WithMethod) Do() {}

// Embedded - only embedded in UnusedEmbedder
type Embedded struct {
	ID int
}

// UnusedEmbedder - embeds Embedded but never used
type UnusedEmbedder struct {
	Embedded
}

func main() {
	var _ Used
}
