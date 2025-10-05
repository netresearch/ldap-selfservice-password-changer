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

func MinNumbersInString(value string, amount uint) bool {
	return countRunesWhere(value, func(c rune) bool {
		return c >= '0' && c <= '9'
	}) >= amount
}

func MinSymbolsInString(value string, amount uint) bool {
	return countRunesWhere(value, func(c rune) bool {
		return (c >= '!' && c <= '/') || (c >= ':' && c <= '@') || (c >= '[' && c <= '`') || (c >= '{' && c <= '~')
	}) >= amount
}

func MinUppercaseLettersInString(value string, amount uint) bool {
	return countRunesWhere(value, func(c rune) bool {
		return c >= 'A' && c <= 'Z'
	}) >= amount
}

func MinLowercaseLettersInString(value string, amount uint) bool {
	return countRunesWhere(value, func(c rune) bool {
		return c >= 'a' && c <= 'z'
	}) >= amount
}
