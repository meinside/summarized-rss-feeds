// exaone.go

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	authorizer "github.com/meinside/exaone-authorizer"
)

const (
	// exaonePlaceholderAPIKey is given to `SetExaoneAPIKeys` with an EXAONE
	// account: the authorizer's transport replaces it with the current key on
	// every request, so it only makes up the combo (and names its cooldowns).
	exaonePlaceholderAPIKey = "exaone-account"

	// login + key issuance takes a few seconds; the authorizer bounds a
	// renewal by 30 seconds on its own
	exaoneInitialKeyTimeout = 1 * time.Minute
)

// newExaoneHTTPClient returns an http client which sets the api key of given
// EXAONE account on requests, issuing and renewing the key as needed.
//
// It gets a key before returning, so that a wrong password or a locked
// account stops the start, instead of every text summary failing (and being
// cached as failed). With a valid key in `key_file`, it does not log in.
//
// NOTE: each start may log in once, and repeated failed logins lock the
// account: limit automatic restarts (see README.md).
func newExaoneHTTPClient(ctx context.Context, account configExaoneAccount) (*http.Client, error) {
	a, err := authorizer.New(
		authorizer.Credentials{Email: account.Email, Password: account.Password},
		authorizer.WithStore(authorizer.FileStore{Path: account.KeyFile}),
		authorizer.WithErrorHandler(func(err error) {
			log.Printf("# failed to renew exaone api key: %s", err)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create exaone authorizer: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, exaoneInitialKeyTimeout)
	defer cancel()

	if _, err := a.APIKey(ctx); err != nil {
		return nil, fmt.Errorf("failed to get exaone api key: %w", err)
	}

	return &http.Client{Transport: a.Transport(nil)}, nil
}
