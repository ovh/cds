package objectstore

type Object interface {
	GetName() string
	GetPath() string
}
