## Description

The goal of this test is to check that we can increase the cluster size.

So the test flow is:

1. create a cluster with 2 masters and 2 workers, checking we have 2 masters and 2 workers.
2. add a master, checking we have 3 masters in total.
3. flush all the tokens.
4. add a worker, checking that a new token is created and we have 3 workers in total.
5. remove a master, checking we have 2 masters now
6. remove a worker , checking we have 2 workers now

All these checks are performed by running `kubect get nodes`.

