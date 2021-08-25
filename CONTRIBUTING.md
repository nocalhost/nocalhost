# Contributing to Nocalhost
Welcome to [report Issues](https://github.com/nocalhost/nocalhost/issues) or [pull requests](https://github.com/nocalhost/nocalhost/pulls). It's recommended to read the following Contributing Guide first before contributing. 

## Issues
We use issues to track public bugs and feature requests.

### Search Known Issues First
Please search the existing issues to see if any similar issue or feature request has already been filed. You should make sure your issue isn't redundant.

### Reporting New Issues
If you open an issue, the more information the better. Such as detailed description, screenshot or video of your problem, logcat or code blocks for your crash.

## Pull Requests
We strongly welcome your pull request to make Nocalhost better. 

### Branch Management
There are three main branches here:

1. `main` branch.
	1. It is the latest (pre-)release branch. We use `main` for tags, with version number `v0.4.10`, `v0.4.12` ...
	2. **Don't submit any PR on `main` branch.**
2. `dev` branch. 
	1. It is our stable developing branch. After full testing, `dev` will be merged to `main` branch for the next release.
	2. **You are recommended to submit bugfix or feature PR on `dev` branch.**

Normal bugfix or feature request should be submitted to `dev` branch. After full testing, we will merge them to `main` branch for the next release. 

### Make Pull Requests
The code team will monitor all pull request, we run some code check and test on it. After all tests passed, we will accecpt this PR. But it won't merge to `main` branch at once, which have some delay.

Before submitting a pull request, please make sure the followings are done:

1. Fork the repo and create your branch from `main`.
2. Update code or documentation if you have changed APIs.
3. Add the copyright notice to the top of any new files you've added.
4. Check your code lints and checkstyles.
5. Test and test again your code.
6. Now, you can submit your pull request on `dev` branch.

## License
By contributing to Nocalhost, you agree that your contributions will be licensed
under its [Apache 2.0 LICENSE](./LICENSE)
