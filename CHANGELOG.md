# Changelog

* Fixed missing documentation for CircleCI plugin.
* Renamed iam plugin to `github.com/zostay/garotate/pkg/plugin/aws/iam/user/access`
* Renamed circleci plugin to `github.com/zostay/garotate/pkg/plugin/circleci/project/env`
* Renamed github plugin to `github.com/zostay/garotate/pkg/plugin/github/action/secret`

## v0.1-alpha1 Sun May  8 01:55:08 2022

* garotate tool for managing secret rotation and storage updates.
* IAM rotation plugin for handling AWS IAM user access key rotations.
* Github storage plugin for storing github action secrets.
* CircleCI storage plugin for storing CircleCI project environment variables.
