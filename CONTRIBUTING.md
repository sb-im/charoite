# Contributing

## Git workflow

To check our code to work on, please refer to [Gitflow Workflow](https://www.atlassian.com/git/tutorials/comparing-workflows/gitflow-workflow).

You should not touch these two branches directly below, but pull a request instead:

* `master`: stable branch.
* `develop`: default actively developing branch.

When your working branch is merged into `develop` branch, it's recommended to delete it soon. And every time you start a new feature branch, you should checkout out it from the latest `develop` branch. Thus you don't have to manage potential conflicts between those old branches and `develop` branch giving you a clean starting point every time.

We also encourage strict code reviewing. An high quality code base demands all members careful maintenances.
