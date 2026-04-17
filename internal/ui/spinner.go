package ui

import "charm.land/huh/v2/spinner"

// WithSpinner runs fn while displaying an inline spinner with the given message.
func WithSpinner[T any](message string, fn func() (T, error)) (T, error) {
	var result T
	var fnErr error
	err := spinner.New().
		Title(message).
		Action(func() {
			result, fnErr = fn()
		}).
		Run()
	if err != nil {
		return result, err
	}
	return result, fnErr
}
