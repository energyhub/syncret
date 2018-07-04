[![Build Status](https://travis-ci.com/energyhub/syncret.svg?token=6cjLqNNpxhcANoBNSgPt&branch=master)](https://travis-ci.com/energyhub/syncret)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/59d3c3383b80449cbce990aba07ea929)](https://www.codacy.com?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=energyhub)
[![Codacy Badge](https://api.codacy.com/project/badge/Coverage/59d3c3383b80449cbce990aba07ea929)](https://www.codacy.com?utm_source=github.com&utm_medium=referral&utm_content=energyhub/syncret&utm_campaign=Badge_Coverage)
[![Go Report Card](https://goreportcard.com/badge/github.com/energyhub/syncret)](https://goreportcard.com/report/github.com/energyhub/syncret)

# syncret

Sync encrypted secrets and their metadata from the local file system to AWS parameter store


## Example:
Given the following file structure:
```
secrets
|_ prod
   |_my-service
        |_DB_URL.gpg
        |_DB_URL.pattern
        |_DB_URL.description
        |_SECRET_KEY.gpg
        |_SECRET_KEY.pattern
        |_SECRET_KEY.description
```

Basic decryption logic on path in a `decrypt.sh` like:
```bash
#!/usr/bin/env bash

set -e

gpg --decrypt ${1}
```

The following command will print all the metadata (not the values) for the matching secrets:

```bash
SYNCRET_DECRYPT=decrypt.sh syncret -prefix secrets/ secrets/prod/my-service/*.gpg
```

And this command will actually install the secrets in AWS:

```bash
SYNCRET_DECRYPT=decrypt.sh syncret -commit -prefix secrets/ secrets/prod/my-service/*.gpg
```

They'll be accessible within the parameter store as:
```
prod/my-service/DB_URL
prod/my-service/SECRET_KEY
```

## decryption logic

Any encryption scheme can be swapped out; only constraint is that `SYNCRET_DECRYPT` be a command on your path that takes as its first argument the file to decrypt and spits it out onto stdout.

## Intended use case

When used with version tracking as a push hook, `syncret` can provide continuous (and secure) deployment of secrets.

The following command installs any modified or added secrets in the `secrets` directory:

```bash
git diff --diff-filter=d --name-only ${SHA_1} ${SHA_2} -- secrets/ | syncret -prefix secrets/
```
