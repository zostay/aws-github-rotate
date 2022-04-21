# garotate

This project provides tooling for automated secret rotation and
disablement. The rotation of secrets is an important task to perform for service
accounts and other related assets, especially in the cloud, in situations where
your provided can't provide such services for you (either because they just
don't or because you need to use the service in such a way that makes those
services unavailable). 

# Getting Started

Currently, this tool does not provide much in the way of help for deployment.
You will need to have Golang installed at least. Installing it from Go just
requires:

```bash
go install github.com/zostay/garotate@latest
```

To install from source:

```bash
git clone git@github.com:zostay/garotate.git
cd garotate
make test && make install
```

# Configuration

Configuration of garotate requires a YAML configuration file and environment
variables to configure the plugins.

## Configuration File

Here's an example configuration file:

```yaml
---
# plugins lists the configurations to use for rotation, disablement, and
# storage. For now, every configuration must define this section exactly like
# this. The names "github" and "IAM" could be changed, but nothing else. These
# two plugins must be configured exactly this way.
plugins:
  github:
    package: github.com/zostay/garotate/pkg/plugin/github
  IAM:
    package: github.com/zostay/garotate/pkg/plugin/iam

# The rotations section configures rotation policies. Each item in the list has
# the following keys:
# 
# client: This names the plugin to use, which must match the name in the plugins
#   section.
# rotate_after: The duration setting that determines how long to wait before
#   rotating the secret. The first run of the rotation tool after this amount of
#   time has passed since last rotation will trigger rotation.
# secret_set: This is the list of secrets that will be rotated according to this
#   policy.
rotations:
  - client: IAM
    rotate_after: "168h"
    secret_set: main

# The disablements section configures disablement policies. Each item in the
# list has the following keys:
#
# client: This names the plugin to use, which must match the name in the plugins
#   section.
# disable_after: The duration setting that determines how long to wait before
#   disabling the secret. The first run of the disablement tool after this
#   amount of time has passed since the secret was created will trigger
#   disablement. You will want this to be longer than the rotation policy time
#   unless you want inactive secrets to be disabled immediately after rotation.
# secret_set: This is the list of secrets that will be rotated according to this
#   policy.
disablements:
  - client: IAM
    disable_after: "216h"
    secret_set: main

# The secret_sets section configures the list of secrets that should have a
# policy applied to them.
#
# Each secret item in the list of secrets must have the following keys:
# 
# secret: The name of the secret to change, whatever names accounts that can be
#   rotated in the plugin. For AWS, this is IAM user name.
# storages: This lists configuration for each of the places that need to be
#   updated after the secret is rotated.
#
# Each storage item in the list of storages must have the following keys:
#
# storage: This is the name of the storage plugin to use. This must exactly
#   match the name of a storage plugin defined in the plugins section.
# name: This is the name of the service that will be receiving a fresh copy of
#   the rotated secret following rotation. This is whatever value the plugin
#   needs. For github, this is the github project name in owner/repo form.
# keys: This is a map that remaps the keys provided by the rotation plugin to
#   the keys to use when storing. The AWS plugin provides two keys,
#   "AWS_ACCESS_KEY_ID" and "AWS_SECRET_ACCESS_KEY". If no keys section is
#   provided, then the keys used are the keys provided by the rotation plugin.
secret_sets:
  - name: main
    secrets:
      - secret: ***REMOVED***
        storages:
          - storage: github
            name: zostay/periodic-s3-sync
            keys:
              AWS_ACCESS_KEY_ID: access_key
              AWS_SECRET_ACCESS_KEY: secret_key
      - secret: ***REMOVED***
        storages:
          - storage: github
            name: zostay/postfix
```

## AWS Plugin Configuration

You must provide AWS configuration using the usual means. This can mean files in
an `~/.aws` folder as used by the AWS CLI or environment variables to provide
the required credentials.

See the [Specifying
Credentials](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials)
section of the Go SDK for full details.

For the IAM plugin to work, these credentials must provide garotate with the
following permissions:

* iam:ListAccessKeys
* iam:CreateAccessKey
* iam:DeleteAccessKey
* iam:UpdateAccessKey

## Github Plugin Configuration.

You must provide a `GITHUB_TOKEN` with `repo` permissions for the github plugin
to work.

Github provides instructions on [creating a personal access
token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token).

# Supported Plugins

Currently, the service supports these plugins:

* Rotation of [AWS IAM users](https://github.com/zostay/garotate/pkg/plugin/iam)
* Storage in [github action secrets](https://github.com/zostay/garotate/pkg/plugin/github)

The plugins are divided into three types, rotation, disablement, and storage.
Typically, the rotation and disablement plugins are going to be the same plugin.
They're split up logically because the process for each operation is slightly
different and the policies that operate on these rotations is different.

The terms plugin and client are almost interchangeable here. Generally, though,
the term plugin refers to the implementation and client refers to the interface.

## Rotation Clients

Rotation clients are responsible for rotating each configured secret. They must
provide the following capabilities:

* Checking the timestamp of the most recent rotation of a given secret.
* Performing the rotation of a secret on request and returning all secret
  details associated with the newly rotated secret.

## Disablement Clients

Disablement clients are responsible for disabling the inactive secrets
associated with an account. Each disablement client must provide the following
capabilities:

* Checking the timestamp of the newst inactive secret associated with an
  account.
* Performing the disablement of all inactive secrets associated with an acount.

## Storage Clients

Storage clients are responsible for storing freshly rotated secrets in some
client-side store. Each storage client must provide the following capabiities:

* Report on the last updated timestamp for each secret associated with a rotated
  account.
* Replace all the secrets associated with a rotated account.

## Rotation/Disablement Plugins

### AWS IAM Users

The AWS IAM users plugin provides an implementation of both the rotation and
disablement clients for rotating AWS IAM user accounts.

## Storage Plugins

### Github Action Secrets

The github action secrets plugin provides an implementation of the storage
client for storing the key associated with rotated accounts.

# The Origin Story

The original use case for this was to help with AWS IAM service accounts that I
provide to some of my Github projects. I have created IAM user accounts,
generate AWS access keys for those accounts, and then save those secrets in the
action secret store for each project. Then the github actions for those projects
can make use of those secrets to perform operations on my AWS account. However,
very shortly, all of my access keys were being flagged as being old, so I wanted
a tool to perform rotation.

I could have found some tool that already does that, but a quick search didn't
find one. (It waas so quick, I might not have read the results or maybe even hit
the search button in DuckDuckGo. I don't remember at this point.) It seemed like
a fun project to do while I was between jobs. It took a bit longer than expected
(of course), so I'm just finishing up writing this a month into the new job.

# COPYRIGHT

Copyright 2022 Andrew Sterling Hanenkamp

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
