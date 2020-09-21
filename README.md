<!--

    TODO:

    - Add the project to the CircleCI:
      https://circleci.com/setup-project/gh/giantswarm/apptest

    - Change the badge (with style=shield):
      https://circleci.com/gh/giantswarm/apptest/edit#badges
      If this is a private repository token with scope `status` will be needed.

    - Update CODEOWNERS file according to the needs for this project

    - Run `devctl replace -i "apptest" "$(basename $(git rev-parse --show-toplevel))" *.md`
      and commit your changes.

    - If the repository is public consider adding godoc badge. This should be
      the first badge separated with a single space.
      [![GoDoc](https://godoc.org/github.com/giantswarm/apptest?status.svg)](http://godoc.org/github.com/giantswarm/apptest)

-->
[![GoDoc](https://godoc.org/github.com/giantswarm/apptest?status.svg)](http://godoc.org/github.com/giantswarm/apptest) [![CircleCI](https://circleci.com/gh/giantswarm/apptest.svg?&style=shield)](https://circleci.com/gh/giantswarm/apptest)

# apptest

Go library for using the Giant Swarm app platform in integration tests.
