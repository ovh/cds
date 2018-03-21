package gorpmapping

import "reflect"

func interfaceToValue(i interface{}) reflect.Value {
	var f reflect.Value
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		f = reflect.ValueOf(i).Elem()
	} else {
		f = reflect.ValueOf(i)
	}

	if f.Kind() == reflect.Interface {
		if reflect.ValueOf(f.Interface()).Kind() == reflect.Ptr {
			f = reflect.ValueOf(f.Interface()).Elem()
		} else {
			f = reflect.ValueOf(f.Interface())
		}
	}
	return f
}
