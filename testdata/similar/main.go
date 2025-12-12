package main

// Test case 1: Exact duplicates
type UserA struct {
	ID    int
	Name  string
	Email string
}

type UserB struct {
	ID    int
	Name  string
	Email string
}

// Test case 2: Subset relationship (embedding candidate)
type BaseEntity struct {
	ID        int
	CreatedAt int64
	UpdatedAt int64
}

type Product struct {
	ID        int
	CreatedAt int64
	UpdatedAt int64
	Name      string
	Price     float64
}

// Test case 3: Same names, different types
type ConfigA struct {
	Host    string
	Port    int
	Timeout int
}

type ConfigB struct {
	Host    string
	Port    string // different type
	Timeout int64  // different type
}

// Test case 4: Partial overlap (mergeable)
type RequestA struct {
	Method  string
	Path    string
	Headers map[string]string
}

type RequestB struct {
	Method string
	Path   string
	Body   []byte
}

// Test case 5: No similarity (should not be reported with default threshold)
type Unrelated1 struct {
	Foo int
	Bar string
}

type Unrelated2 struct {
	Baz  float64
	Qux  bool
	Quux int
}

// Test case 6: Single field structs (should be filtered by default MinFields=2)
type Single1 struct {
	X int
}

type Single2 struct {
	X int
}

func main() {}
