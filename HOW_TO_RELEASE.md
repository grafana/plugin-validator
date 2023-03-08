# How to release a new version

**IMPORTANT:** Do not push release tags manually

* Go to Actions -> Bump version
* Click on "Run workflow"
* Keep branch: main
* Select the type of new version (patch, minor, major)
* Run workflow

>External contributors note: For security purposes, externals contributors don't usually have permissions to create new releases.

This will trigger a series of github actions to build a new github release and automatically push to npm


# Github or NPM tokens expiration

Github and NPM tokens are required to auto-publish. These tokens eventually expire. If there's an error pushing tags to github or updates to npm make sure to review the tokens expiration.
