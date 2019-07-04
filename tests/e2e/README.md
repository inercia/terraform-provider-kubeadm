## End-to-end (e2e) tests

The e2e tests here are simple scripts based on invoking `terraform`
in the testing environment with different variables that will
drive the test.

The environment is specified in the `$E2E_ENV` variable, and
is currently one of the directories in [docs/examples](../docs/examples).
By default, we will use the `DnD` environment.

The variables will be things like number of _masters/workers_
(specified in the  `master_count`/`worker_count` variables). When running
in Travis, we will add any variables defined in `$E2E_ENV/ci.tfvars`.

Checkout the subdirectories for more details on the current tests suites...
