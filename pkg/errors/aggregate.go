// Package errors provides the tools for gathering errors for the processes in
// this program which keep working when there are errors.
package errors

import "strings"

// Aggregate groups a list of errors together.
type Aggregate struct {
	errlist []error
}

// NewAggregate returns an Aggregate containing the given list of errors.
func NewAggregate(errlist []error) *Aggregate {
	return &Aggregate{errlist}
}

// Error returns the combined error message for the Aggregate.
func (a *Aggregate) Error() string {
	msg := new(strings.Builder)
	first := true
	for _, err := range a.errlist {
		if !first {
			msg.WriteString("; ")
		}
		msg.WriteString(err.Error())
		first = false
	}
	return msg.String()
}

// Errors returns the individual errors which make up the aggregate.
func (a *Aggregate) Errors() []error {
	return a.errlist
}
