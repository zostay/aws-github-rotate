// Package disable provides the business logic for managing the disablement of
// old secrets that have been rotated. In a typical secret rotation use-case, at
// least two active secrets are maintained at the point of rotation to avoid
// causing an outage for any running process using the current secret. Then, a
// followup process will disable/delete the old secret after the new secret has
// been established. This pakcage manages the disablement process.
package disable
