package internal

func IsEnable(e *bool) bool {
	if e == nil {
		return true
	}
	return *e
}
