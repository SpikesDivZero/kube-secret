# kube-secret

The primary purpose of kube-secret is to make it easier to edit kube secret files, where the data is base64 encoded.

Inspired by [lbolla/kube-secret-editor](https://github.com/lbolla/kube-secret-editor), with a few notable differences:

* YAML key ordering is preserved. If you're editing a secret file stored in a git repository, then keeping the keys in
  the right order matters. Stray changes like swapping keys around create more cognitive load on the reviewer, and I'd
  rather save that workload for important things. Not to mention, having a clean git history is always nice!

* Compiled Go binary, so upgrades to the language interpreter won't break the tool.

## Overview

**Installation:** Assuming your environment is configured for go development (PATH included), then kube-secret can be
installed via:

    go install github.com/spikesdivzero/kube-secret@latest

**Usage:** The most common use case is to edit a secret file and write it back in a single pass. To do so, you'll run:

    kube-secret edit test-data/fake-secret.yaml

When invoked, it'll do the following:

* Decode the secrets from base64 to human-friendly strings.
* Invoke `$EDITOR` on the decoded secret file
* Encode the updated secrets as base64 again to keep k8s happy.

**Usage w/ kubectl:** To use kube-secret with `kubectl`, you'll want to set up a wrapping alias, akin to:

    alias kedit-secret="KUBE_EDITOR='kube-secret' kubectl edit secret"

Editing a secret becomes as easy as "kedit-secret secrets/my-secret".

You might have noticed that KUBE_EDITOR in that alias doesn't include the "edit" verb. If kube-secret is invoked with
a filename as the first argument, then it assumes the intended usage was "kube-secret edit".

## Notes

* If EDITOR is not configured in your user's environment, then kube-secret defaults to `vi`.
* Any time you're working with secrets, your environment should be configured for full-disk encryption.
