package tat

// Version of Tat
// One Line for this, used by release.sh script
// Keep "const Version on one line"
const Version = "5.2.1"

const (
	// TatHeaderUsername is Tat_username header
	TatHeaderUsername = "Tat_username"
	// TatHeaderPassword is Tat_password header
	TatHeaderPassword = "Tat_password"
	// TatHeaderXTatRefererLower contains tat microservice name & version "X-TAT-FROM"
	TatHeaderXTatRefererLower = "X-Tat-Referer"
)

// ArrayContains return true if element is in array
func ArrayContains(array []string, element string) bool {
	for _, cur := range array {
		if cur == element {
			return true
		}
	}
	return false
}

// ItemInBothArrays return true if an element is in both array
func ItemInBothArrays(arrayA, arrayB []string) bool {
	for _, cura := range arrayA {
		for _, curb := range arrayB {
			if cura == curb {
				return true
			}
		}
	}
	return false
}

//CacheableCriteria must return strnig slice describing the redis key
type CacheableCriteria interface {
	CacheKey() []string
}
