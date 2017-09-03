package worker

import (
	"fmt"
)

// MultiError is returned by batch operations when there are errors with
// particular elements. Errors will be in a one-to-one correspondence with
// the input elements; successful elements will have a nil entry.
type MultiError []error

func (m MultiError) Error() string {
	s, n := "", 0
	for _, e := range m {
		if e != nil {
			s = s + fmt.Sprintf("\n\t - %s", e.Error())
			n++
		}
	}
	if n > 1 {
		s = "Multi errors: " + s
	}
	return s
}
