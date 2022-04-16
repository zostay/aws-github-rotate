package disable

import (
	"context"
	"time"

	"github.com/zostay/aws-github-rotate/pkg/secret"
)

// Client defines the interface that any plugin that wishes to perform
// disablement must implement. It provides means for identifying the client,
// detecting when a configured secret is ready for disablement, and the method
// for performing disablement.
type Client interface {
	// Name should return a string that clearly identifies the plugin to the
	// administrator and is attached to log and error messages.
	Name() string

	// LastUpdated must return the timestamp when the newest inactive secret was
	// last updated. Usually this will be the creation data of an access token
	// or other piece of data.
	//
	// The context provides a logger via the
	// github.com/zostay/aws-github-rotate/pkg/config package. It may also be
	// used for timeouts.
	//
	// The secret.Info describes the secret that is being checked for
	// disablement.
	LastUpdated(context.Context, secret.Info) (time.Time, error)

	// DisableSecret must perform disablement of all inactive secrets associated
	// with the account.
	//
	// The context provides a logger via the
	// github.com/zostay/aws-github-rotate/pkg/config package. It may also be
	// used for timeouts.
	//
	// The secret.Info describes the secret that is being checked for
	// disablement.
	DisableSecret(context.Context, secret.Info) error
}
