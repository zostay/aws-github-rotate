package disable

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

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
