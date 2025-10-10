// Package validators provides password validation functions for character type requirements.
package validators

// countRunesWhere counts runes in the string that satisfy the predicate.
func countRunesWhere(value string, predicate func(rune) bool) uint {
	var counter uint
	for _, c := range value {
		if predicate(c) {
			counter++
		}
	}
	return counter
}

// MinNumbersInString checks if the string contains at least the specified number of numeric digits.
func MinNumbersInString(value string, amount uint) bool {
	return countRunesWhere(value, func(c rune) bool {
		return c >= '0' && c <= '9'
	}) >= amount
}

// MinSymbolsInString checks if the string contains at least the specified number of special characters.
func MinSymbolsInString(value string, amount uint) bool {
	return countRunesWhere(value, func(c rune) bool {
		return (c >= '!' && c <= '/') || (c >= ':' && c <= '@') || (c >= '[' && c <= '`') || (c >= '{' && c <= '~')
	}) >= amount
}

// MinUppercaseLettersInString checks if the string contains at least the specified number of uppercase letters.
func MinUppercaseLettersInString(value string, amount uint) bool {
	return countRunesWhere(value, func(c rune) bool {
		return c >= 'A' && c <= 'Z'
	}) >= amount
}

// MinLowercaseLettersInString checks if the string contains at least the specified number of lowercase letters.
func MinLowercaseLettersInString(value string, amount uint) bool {
	return countRunesWhere(value, func(c rune) bool {
		return c >= 'a' && c <= 'z'
	}) >= amount
}
