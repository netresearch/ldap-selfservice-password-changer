package validators

func MinNumbersInString(value string, amount int) bool {
	counter := 0
	for _, c := range value {
		if c >= '0' && c <= '9' {
			counter++
		}
	}

	return counter >= amount
}

func MinSymbolsInString(value string, amount int) bool {
	counter := 0
	for _, c := range value {
		if (c >= '!' && c <= '/') || (c >= ':' && c <= '@') || (c >= '[' && c <= '`') || (c >= '{' && c <= '~') {
			counter++
		}
	}

	return counter >= amount
}

func MinUppercaseLettersInString(value string, amount int) bool {
	counter := 0
	for _, c := range value {
		if c >= 'A' && c <= 'Z' {
			counter++
		}
	}

	return counter >= amount
}

func MinLowercaseLettersInString(value string, amount int) bool {
	counter := 0
	for _, c := range value {
		if c >= 'a' && c <= 'z' {
			counter++
		}
	}

	return counter >= amount
}
