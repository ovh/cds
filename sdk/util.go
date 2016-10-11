package sdk

// IsInArray Check if the element is in the array
func IsInArray(elt string, array []string) bool {
	for _, item := range array {
		if item == elt {
			return true
		}
	}
	return false
}
