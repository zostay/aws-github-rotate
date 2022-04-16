package disable

import (
	"context"
	"fmt"
	"time"

	"github.com/zostay/garotate/pkg/config"
)

// Manager provides the business logic for detecting whether a secret is old
// enough to require disablement or not and disable those secrets.
type Manager struct {
	client Client

	disableAfter time.Duration

	dryRun bool

	secrets []config.Secret
}

// New constructs a new object to perform password disablement.
func New(
	rc Client,
	disableAfter time.Duration,
	dryRun bool,
	secrets []config.Secret,
) *Manager {
	return &Manager{
		client:       rc,
		disableAfter: disableAfter,
		dryRun:       dryRun,
		secrets:      secrets,
	}
}

// needsDisablement returns true if the secret has an updated date older than
// disableAfter in the past.
func (m *Manager) needsDisablement(
	ctx context.Context,
	s *config.Secret,
) bool {
	logger := config.LoggerFrom(ctx).Sugar()

	updateDate, err := m.client.LastUpdated(ctx, s)
	if err != nil {
		logger.Errorw(
			"got error while checking last update date for disablement; skipping",
			"secret", s.SecretName,
			"client", m.client.Name(),
		)
		return false
	}

	hasNeed := time.Since(updateDate) > m.disableAfter
	if hasNeed {
		logger.Debugw(
			"secret is active and old enough to require disablement",
			"secret", s.SecretName,
			"client", m.client.Name(),
			"now_ts", time.Now(),
			"update_ts", updateDate,
			"disable_after", m.disableAfter,
		)
	}
	return hasNeed
}

// disableSecret checks to see if the secret given requires disablement and
// disables it if it does.
func (m *Manager) disableSecret(ctx context.Context, s *config.Secret) error {
	if !m.needsDisablement(ctx, s) {
		return nil
	}

	if !m.dryRun {
		err := m.client.DisableSecret(ctx, s)
		if err != nil {
			return fmt.Errorf("failed to disable old active secret %q for disabler %q", s.SecretName, m.client.Name())
		}
	} else {
		logger := config.LoggerFrom(ctx).Sugar()
		logger.Infow(
			"dry run: here's where the old secret should get disable",
			"secret", s.Name(),
			"client", m.client.Name(),
		)
	}
	return nil
}

// DisableSecrets examines all the IAM keys and disables any of the
// non-active keys that have surpassed the maxActiveAge.
func (m *Manager) DisableSecrets(ctx context.Context) error {
	logger := config.LoggerFrom(ctx).Sugar()
	for k := range m.secrets {
		s := &m.secrets[k]
		logger.Debugw(
			"examining secret for disablement",
			"secret", s.SecretName,
			"client", m.client.Name(),
		)

		err := m.disableSecret(ctx, s)
		if err != nil {
			logger.Errorw(
				"failed to disable secret",
				"secret", s.SecretName,
				"client", m.client.Name(),
			)
			continue
		}
	}

	return nil
}
