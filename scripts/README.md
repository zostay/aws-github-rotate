# Making Release

In order to cut a release of garotate, perform the following steps:

  1. Run `./scripts/start-release <version>`

     The version number must be in one of the following forms:

     * v#.#
     * v#.#-alpha#
     * v#.#-beta#
     * v#.#-rc#

     The hyphenated forms will result in the prerelease flag being set.

  2. Monitor CI/CD builds to ensure successful builds.

  3. Run `./scripts/finish-release <assets>...`

     Each `<asset>` is the name of a file to add to the release, which has been
     uploaded to the S3 bucket found here:

     `s3://garotate.qubling.cloud/release-<short-version>`

     Where `<short-version>` is the `<version>` without the `v` on the front.

  4. Go to the [release](https://github.com/zostay/garotate/releases) page,
     click on the draft release. Proofread and publish.

# Undoing a Release

If something goes wrong in the middle of a release. Some or all of the following
commands will completely revert the release work:

```bash
# to delete the github release
gh release delete <version>

# to clean the binary files cache
aws s3 rm --recursive s3://garotate.qubling.cloud/release-<short-version>

# to remove the version tag
git tag -d <version>
git push origin :<version>

# to remove the release trigger tag
git tag -d release-<short-version>
git push origin :release-<short-version>
```
