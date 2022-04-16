package rotate

import (
	"context"
	"fmt"
	"time"

	"github.com/zostay/aws-github-rotate/pkg/config"
	"github.com/zostay/aws-github-rotate/pkg/plugin"
	"github.com/zostay/aws-github-rotate/pkg/secret"
)

// Manager provides the business logic for detecting whether secrets in the
// associated Client require rotation. If so, it tells the rotation client to
// perform the rotation. It then notifies all storages of the updated key
// values.
type Manager struct {
	plugins *plugin.Manager

	client Client

	rotateAfter time.Duration

	dryRun bool

	secrets []config.Secret
}

// New constructs a new object to perform password rotation.
func New(
	rc Client,
	rotateAfter time.Duration,
	dryRun bool,
	plugins *plugin.Manager,
	secrets []config.Secret,
) *Manager {
	return &Manager{
		plugins:     plugins,
		client:      rc,
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
	s *config.Secret,
) bool {
	logger := config.LoggerFrom(ctx).Sugar()

	// TODO Make this method safer against transient storage errors that could
	// result in a rotation occurring without all storages being notified. We
	// can't make any ACID guarantees, but the current implementation is not as
	// robust as it could be. This should be changed as follows.
	//
	// Do not short-circuit the decision in the positive until all checks that
	// might make it positive have been made.
	//
	// However, this too has problems. We have to weigh two risks:
	//
	// 1. A misconfigured storage or transient storage error could result in a
	//    rotation being performed, but the storage with the problem is never
	//    updated. This is a good sort of problem as it will likely lead to
	//    loud failures, but bad because it will be the sort of
	//    action-at-distance failure that might not result in an easy fix.
	//
	// 2. A delay in rotation because of storage misconfigurations and errors
	//    might result in a violation of policy because rotation gets delayed
	//    indefinitely if not noticed. Can we fail loudly in a way that forces
	//    notice? (For example, if disabling is on, the active key will
	//    eventually age out, which would cause loud failures, but again of the
	//    action at a distance variety.) Or we could resume rotation if the
	//    storage remains misconfigured/in a transient bad state for too long.
	//
	// How do we balance these risks to ensure meeting our rotation policies and
	// also making sure a tranisent error doesn't cause a storage to miss out on
	// a rotation.

	rotated, err := m.client.LastRotated(ctx, s)
	if err != nil {
		logger.Errorw(
			"got error while checking last rotation date; skipping",
			"secret", s.Name(),
			"client", m.client.Name(),
		)
		return false
	}

	if time.Since(rotated) > m.rotateAfter {
		logger.Debugw(
			"secret is out of date and requires rotation",
			"secret", s.Name(),
			"client", m.client.Name(),
			"now_ts", time.Now(),
			"rotation_ts", rotated,
			"rotate_after", m.rotateAfter,
		)
		return true
	}

	for i := range s.Storages {
		sm := &s.Storages[i]
		store, err := m.findStorage(ctx, sm.StorageClient)
		if err != nil {
			logger.Errorw(
				"got error while loading storage plugin; for safety, rotation will be prevented",
				"secret", s.Name(),
				"store_name", sm.StorageClient,
			)
			return false
		}

		for _, storeKey := range sm.Keys {
			saved, err := store.LastSaved(ctx, sm, storeKey)
			if err != nil {
				logger.Errorw(
					"got error while checking last storage date; for safety, rotation will be prevented",
					"secret", s.Name(),
					"store_name", sm.StorageClient,
					"store_desc", store.Name(),
					"store_key", storeKey,
					"error", err,
				)
				return false
			}

			if saved.Before(rotated) {
				logger.Debugw(
					"secret stored is older than most recent rotation",
					"secret", s.SecretName,
					"client", m.client.Name(),
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

// findStorage returns a constructed storage client instance for the given name
// or an error.
func (m *Manager) findStorage(
	ctx context.Context,
	name string,
) (Storage, error) {
	inst, err := m.plugins.Instance(ctx, name)
	if err != nil {
		return nil, err
	}

	if store, ok := inst.(Storage); ok {
		return store, nil
	}

	return nil, fmt.Errorf("expected storage plugin for client named %q, but got %T instead", name, inst)
}

// remapKeys will rewrite the keys of the incoming map according to the given
// key map. Original keys will be kept as-is unless remapped. A remapped key,
// however, will be removed from the original.
func remapKeys(
	keyMap config.KeyMap,
	secrets secret.Map,
) secret.Map {
	remapSecrets := make(secret.Map, len(secrets))
	for k, v := range secrets {
		if newK, ok := keyMap[k]; ok {
			remapSecrets[newK] = v
		} else {
			remapSecrets[k] = v
		}
	}
	return remapSecrets
}

// rotateSecret rotates a single secret. It checks if the secret needs to be
// rotated by calling needsRotation(). If not, it does nothing further. If so,
// it tells the rotation client to rotate the secret. It then it saves the newly
// minted secret in all configured storage locations.
func (m *Manager) rotateSecret(
	ctx context.Context,
	s *config.Secret,
) error {
	if !m.needsRotation(ctx, s) {
		return nil
	}

	logger := config.LoggerFrom(ctx).Sugar()

	var (
		newSecrets secret.Map
		err        error
	)
	if !m.dryRun {
		newSecrets, err = m.client.RotateSecret(ctx, s)
		if err != nil {
			return fmt.Errorf("RotateSecret(): %w", err)
		}
	} else {
		logger.Infow(
			"dry run: here's where the secret should get rotated",
			"secret", s.Name(),
			"client", m.client.Name(),
		)
	}

	for i := range s.Storages {
		sm := &s.Storages[i]
		store, err := m.findStorage(ctx, sm.StorageClient)
		if err != nil {
			return fmt.Errorf("error while loading storage plugin %q: %w", sm.StorageClient, err)
		}

		remappedSecret := remapKeys(sm.Keys, newSecrets)
		if !m.dryRun {
			err = store.SaveKeys(ctx, sm, remappedSecret)
			if err != nil {
				logger.Errorw(
					"failed to update storage with newly rotated secrets",
					"secret", s.Name(),
					"client", m.client.Name(),
					"store", store.Name(),
					"error", err,
				)
			}
		} else {
			logger.Infow(
				"dry run: here's where the secret should get stored",
				"secret", s.Name(),
				"client", m.client.Name(),
				"store", store.Name(),
			)
		}

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
	logger := config.LoggerFrom(ctx).Sugar()
	for i := range m.secrets {
		s := &m.secrets[i]
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
