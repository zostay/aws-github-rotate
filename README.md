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

TODO

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
