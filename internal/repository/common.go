package repository

type scanFunc func(dest ...any) error

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
