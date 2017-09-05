package supervisor

import (
	"fmt"
)

// MultiError is returned by batch operations when there are errors with
// particular elements. Errors will be in a one-to-one correspondence with
// the input elements; successful elements will have a nil entry.
type MultiError []error

func (m MultiError) Error() string {
	s, n := "", 0
	var lastErr error
	for _, e := range m {
		if e != nil {
			lastErr = e
			s = s + fmt.Sprintf("\n\t - %s", e.Error())
			n++
		}
	}
	if n == 1 {
		return lastErr.Error()
	}
	return "Multi errors: " + s
}
