package gorpmapping

// TableMapping represents a table mapping with gorp
type TableMapping struct {
	Target        interface{}
	Name          string
	AutoIncrement bool
	Keys          []string
}

// New initialize a TableMapping
func New(t interface{}, n string, b bool, k ...string) TableMapping {
	return TableMapping{t, n, b, k}
}

// Mapping is the global var for all registered mapping
var Mapping []TableMapping

//Register intialiaze gorp mapping
func Register(m ...TableMapping) {
	for _, t := range m {
		Mapping = append(Mapping, t)
	}
}
