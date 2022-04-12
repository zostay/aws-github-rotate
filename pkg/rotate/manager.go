package rotate

import (
	"context"
	"fmt"
	"time"

	"github.com/zostay/aws-github-rotate/pkg/config"
)

// Manager provides the business logic for detecting whether secrets in the
// associated Client require rotation. If so, it tells the rotation client to
// perform the rotation. It then notifies all storages of the updated key
// values.
type Manager struct {
	client Client
	stores Storages

	rotateAfter time.Duration

	dryRun bool

	secrets []*Secret
}

// New constructs a new object to perform password rotation.
func New(
	rc Client,
	scs Storages,
	rotateAfter time.Duration,
	dryRun bool,
	secrets []*config.Secret,
) *Manager {
	ss := make([]*Secret, len(secrets))
	for i, s := range secrets {
		ss[i] = NewSecret(s)
	}

	return &Manager{
		client:      rc,
		stores:      scs,
		rotateAfter: rotateAfter,
		dryRun:      dryRun,
		secrets:     secrets,
	}
}

// needsRotation returns true if either of two conditions is true:
//
// 1. LastSaved() value of any secret key associated with this project in any
//    Storage is older than rotateAfter.
// 2. LastRotate() value of the project is newer than any LastSaved() value of
//    any secret key associated with this project in any Storage.
//
// Otherwise, this returns false.
func (m *Manager) needsRotation(
	ctx context.Context,
	s *Secret,
) bool {
	logger := config.LoggerFrom(ctx).Sugar()

	rotated, err := m.client.LastRotated(ctx, s)
	if err != nil {
		logger.Errorw(
			"got error while checking last rotation date; skipping",
			"secret", s.Secret(),
			"client", client.Name(),
		)
		return false
	}

	if time.Since(rotated) > m.rotateAfter {
		logger.Debugw(
			"secret is out of date and requires rotation",
			"secret", s.Seret(),
			"client", client.Name(),
			"now_ts", time.Now(),
			"rotation_ts", rotated,
			"rotate_after", m.rotateAfter,
		)
		return true
	}

	for _, si := range secret.Storages {
		store := m.findStorage(si.Storage())
		for _, storeKey := range si.Keys {
			saved, err := store.LastSave(ctx, si, storeKey)
			if err != nil {
				logger.Errorw(
					"got error while checking last storage date; skipping",
					"secret", s.Name(),
					"store", store.Name(),
					"store_key", storeKey,
					"error", err,
				)
				continue
			}

			if saved < rotated {
				loger.Debugw(
					"secret stored is older than most recent rotation",
					"secret", s.Secret(),
					"client", client.Name(),
					"storage", store.Name(),
					"rotation_ts", rotated,
					"saved_ts", saved,
				)
				return true
			}
		}
	}

	return false
}

// rotateSecret rotates a single secret. It checks if the secret needs to be
// rotated by calling needsRotation(). If not, it does nothing further. If so,
// it tells the rotation client to rotate the secret. It then it saves the newly
// minted secret in all configured storage locations.
func (m *Manager) rotateSecret(ctx context.Context, s *Secret) error {
	if !m.needsRotation(ctx, s) {
		return nil
	}

	newSecrets, err := m.client.RotateSecret(ctx, s)
	if err != nil {
		return fmt.Errorf("RotateSecret(): %w", err)
	}

	for _, si := range s.Storages {
		store := m.findStorage(si.Storage())
		remappedSecret := m.remapKeys(si.Keys(), newSecrets)
		err := store.SaveKeys(ctx, si, remappedSecret)
		if err != nil {
			logger.Errorw(
				"failed to update storage with newly rotated secrets",
				"secret", s.Name(),
				"client", m.client.Name(),
				"store", store.Name(),
				"error", err,
			)
		}
	}
	err = r.updateGithub(ctx, ak, sk, p)
	if err != nil {
		return fmt.Errorf("updateSecrets(): %w", err)
	}

	return nil
}

// RotateSecrets goes through all the configured secrets, determines which
// require rotation, either because the time since the last rotation is greater
// than the configured maximum duration or because one of the storages has a
// copy of the secret that is older than the last rotation.
//
// Each rotation that is needed is performed and all storages associated with
// each rotation are updated.
func (m *Manager) RotateSecrets(ctx context.Context) error {
	logger := config.LoggerFrom(ctx)
	for _, s := range m.Secrets {
		logger.Debugw(
			"examining secret for rotation",
			"secret", s.Name(),
			"client", m.client.Name(),
		)

		err := m.rotateSecret(ctx, s)
		if err != nil {
			return fmt.Errorf("failed to rotate secret: %w", err)
		}
	}

	return nil
}
