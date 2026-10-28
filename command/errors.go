package command

import (
	"errors"
	"fmt"
)

// ErrHelp is returned inside a run once help has been printed, to stop it.
// Run and RunContext turn it into nil.
var ErrHelp = errors.New("help requested")

// UsageError reports input that the command line rejected: an unknown
// command or flag, a missing argument, a value of the wrong type and so on.
type UsageError struct {
	// Command is the full path of the command that rejected the input, such
	// as "app db migrate".
	Command string
	Err     error
}

func (e *UsageError) Error() string { return e.Err.Error() }

func (e *UsageError) Unwrap() error { return e.Err }

// UsageErrorf returns a *UsageError for a handler that finds a problem with
// its input that gear doesn't check itself. Its Command is filled in when the
// handler returns it.
func UsageErrorf(format string, a ...any) error {
	return &UsageError{Err: fmt.Errorf(format, a...)}
}

// ExitCode returns the conventional exit status for an error from Run: 0 for
// nil, 2 for a *UsageError and 1 for anything else.
func ExitCode(err error) int {
	var usage *UsageError
	switch {
	case err == nil || errors.Is(err, ErrHelp):
		return 0
	case errors.As(err, &usage):
		return 2
	default:
		return 1
	}
}
