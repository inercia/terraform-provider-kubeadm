## End-to-end (e2e) tests

The _e2e_ tests here are simple scripts based on invoking `terraform`
in a testing environment with different variables that will
drive the test.

The environment is specified in the `$E2E_ENV` variable, and
is currently one of the directories in [docs/examples](../docs/examples).
By default, we will use the `DnD` environment. You can run the `e2e` test in
a different environment with something like

```console
$ make e2e E2E_ENV=`pwd`/docs/examples/aws
```

The Terraform cluster is left intact in case of an error for
forensics purposes. You can destroy it with:

```console
$ make e2e-cleanup E2E_ENV=`pwd`/docs/examples/aws
```

Checkout the subdirectories for more details on the current tests suites...

## Variables

The variables that drive these tests will be things like number of
_masters/workers_ (specified in the  `master_count`/`worker_count` variables).
When running in Travis, we will add any variables defined in `$E2E_ENV/ci.tfvars`.

Some other vars are:

* `E2E_ENV`: an absolute directory with the tests environment

* `E2E_CLEANUP`: clean up the environment (ie, `terraform destroy`) after failing a tests suite.

## Logs

A debug log from Terraform is left at `$E2E_ENV/terraform.log`.
