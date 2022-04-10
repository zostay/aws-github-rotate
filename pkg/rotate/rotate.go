package rotate

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
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

	projects []*Project
}

// New constructs a new object to perform password rotation.
func New(
	rc Client,
	scs Storages,
	rotateAfter time.Duration,
	dryRun bool,
	projects []*config.Project,
) *Manager {
	ps := make([]*Project, len(projects))
	for i, p := range projects {
		ps[i] = NewProject(p)
	}

	return &Manager{
		client:      rc,
		stores:      scs,
		rotateAfter: rotateAfter,
		dryRun:      dryRun,
		projects:    ps,
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
) (bool, error) {
	logger := config.LoggerFrom(ctx).Sugar()

	rotated, err := m.client.LastRotated(ctx, s)
	if err != nil {
		logger.Errorw(
			"got error while checking last rotation date; skipping",
			"secret", s.Secret(),
			"client", client.Name(),
		)
		continue
	}

	if time.Since(rotated) > m.rotateAfter {
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
				)
				continue
			}

			if saved < rotated {
				return true
			}
		}
	}

	return false
}

// rotateSecret rotates a single project's secret in IAM and then updates the
// github action secret keys with the newly minted access key and secret key.
func (r *Rotate) rotateSecret(ctx context.Context, p *Project) error {
	needed, err := r.NeedsRotation(ctx, p)
	if err != nil {
		return err
	} else if !needed {
		return nil
	}

	ak, sk, err := r.rotateAWSSecret(ctx, p)
	if err != nil {
		return fmt.Errorf("rotateSecretIam(): %w", err)
	}

	err = r.updateGithub(ctx, ak, sk, p)
	if err != nil {
		return fmt.Errorf("updateSecrets(): %w", err)
	}

	return nil
}

// RotateSecrets goes through all the projects, determines which have
// outdated keys (i.e., they are older than maxAge) or a mismatch between IAM
// information and github information and performs rotation on those services.
// All github services should have working keys after this operation is
// performed.
func (r *Rotate) RotateSecrets(ctx context.Context) error {
	for k := range r.Projects {
		if r.verbose {
			fmt.Printf("Consider for rotation, project %s\n", r.Projects[k].Name)
		}
		err := r.rotateSecret(ctx, r.Projects[k])
		if err != nil {
			return fmt.Errorf("failed to rotate secret: %w", err)
		}
	}

	return nil
}

// disableAWSSecret examines the given project to see if the oldestKey is older
// than the DisableAfter time. If it is older, then the key is disabled in IAM.
// If not, then it is left alone.
func (r *Rotate) disableAWSSecret(ctx context.Context, p *Project) error {
	okey, _, err := r.getAccessKeys(ctx, p)
	if err != nil {
		return fmt.Errorf("failed to retrieve key information: %w", err)
	}

	createDate := aws.TimeValue(okey.CreateDate)
	needsDisabled := time.Since(createDate) > r.rotateAfter+r.disableAfter
	if !needsDisabled {
		return nil
	}

	if r.verbose && needsDisabled {
		fmt.Printf(" - Secret updated %v has not been updated in more than %v\n", p.SecretUpdatedAt, r.rotateAfter+r.disableAfter)
	}

	if r.verbose {
		fmt.Print(" - ")
	}
	fmt.Printf("disabling old IAM account key for project %s\n", p.Name)

	p.ClearAWSKeyCache()
	_, err = r.svcIam.UpdateAccessKey(&iam.UpdateAccessKeyInput{
		AccessKeyId: okey.AccessKeyId,
		Status:      aws.String(iam.StatusTypeInactive),
		UserName:    aws.String(p.User),
	})
	if err != nil {
		return fmt.Errorf("failed to update access key status to inactive: %w", err)
	}

	return nil
}

// DisableOldSecrets examines all the IAM keys and disables any of the
// non-active keys that have surpassed the maxActiveAge.
func (r *Rotate) DisableOldSecrets(ctx context.Context) error {
	for k := range r.Projects {
		if r.verbose {
			fmt.Printf("Consider for disablement, project %s\n", r.Projects[k].Name)
		}
		err := r.disableAWSSecret(ctx, r.Projects[k])
		if err != nil {
			return fmt.Errorf("failed to disable secret: %w", err)
		}
	}

	return nil
}
