## Description

The goal of this test is to check that we can increase the cluster size.

So the test flow is:

1. create a cluster with 2 masters and 2 workers, checking we have 2 masters and 2 workers.
2. add a master, checking we have 3 masters in total.
3. add a worker, checking we have 3 masters in total.

All these checks are performed by running `kubect get nodes`.

