package internal

func Bool(e bool) *bool {
	return &e
}

func IsEnable(e *bool) bool {
	if e == nil {
		return true
	}
	return *e
}
