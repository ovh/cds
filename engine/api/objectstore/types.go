package objectstore

//Object is the interface for stuff needed to be stored in object store
type Object interface {
	GetName() string
	GetPath() string
}
