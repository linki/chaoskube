package util

// StripElements substracts a slice of candidates from a slice of elements
func StripElements(elements []string, candidates ...string) []string {
	if len(candidates) == 0 {
		return elements
	}

	result := []string{}

	for _, element := range elements {
		if !containsElement(candidates, element) {
			result = append(result, element)
		}
	}

	return result
}

// containsElement returns true if candidate is a member of elements
func containsElement(elements []string, candidate string) bool {
	for _, element := range elements {
		if element == candidate {
			return true
		}
	}

	return false
}
