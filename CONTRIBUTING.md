# Contributing to Nocalhost
Welcome to [report Issues](https://github.com/nocalhost/nocalhost/issues) or [pull requests](https://github.com/nocalhost/nocalhost/pulls). It's recommended to read the following Contributing Guide first before contributing.

## Issues
We use issues to track public bugs and feature requests.

### Search Known Issues First
Please search the existing issues to see if any similar issue or feature request has already been filed. You should make sure your issue isn't redundant.

### Reporting New Issues
If you open an issue, the more information the better. Such as detailed description, screenshot or video of your problem, logcat or code blocks for your crash.

## <a name="commit-signing">Signing-off on Commits (Developer Certificate of Origin)</a>

To contribute to this project, you must agree to the Developer Certificate of
Origin (DCO) for each commit you make. The DCO is a simple statement that you,
as a contributor, have the legal right to make the contribution.

See the [DCO](https://developercertificate.org) file for the full text of what you must agree to
and how it works [here](https://github.com/probot/dco#how-it-works).
To signify that you agree to the DCO for contributions, you simply add a line to each of your
git commit messages:

```
Signed-off-by: Tom <tom@example.com>
```

In most cases, you can add this signoff to your commit automatically with the
`-s` or `--signoff` flag to `git commit`. You must use your real name and a reachable email
address (sorry, no pseudonyms or anonymous contributions). An example of signing off on a commit:
```
$ commit -s -m “my commit message w/signoff”
```

### Issue Types

There are 5 types of issues (each with their own corresponding [label](#labels)):

- `question/support`: These are support or functionality inquiries that we want to have a record of
  for future reference. Generally these are questions that are too complex or large to store in the
  Slack channel or have particular interest to the community as a whole. Depending on the
  discussion, these can turn into `feature` or `bug` issues.
- `proposal`: Used for items (like this one) that propose a new ideas or functionality that require
  a larger community discussion. This allows for feedback from others in the community before a
  feature is actually  developed. This is not needed for small additions. Final word on whether or
  not a feature needs a proposal is up to the core maintainers. All issues that are proposals should
  both have a label and an issue title of "Proposal: [the rest of the title]." A proposal can become
  a `feature` and does not require a milestone.
- `feature`: These track specific feature requests and ideas until they are complete. They can
  evolve from a `proposal` or can be submitted individually depending on the size.
- `bug`: These track bugs with the code
- `docs`: These track problems with the documentation (i.e. missing or incomplete)

### Issue Lifecycle

The issue lifecycle is mainly driven by the core maintainers, but is good information for those
contributing to Nocalhost. All issue types follow the same general lifecycle. Differences are noted
below.

1. Issue creation
2. Triage
	- The maintainer in charge of triaging will apply the proper labels for the issue. This includes
	  labels for priority, type, and metadata (such as `good first issue`). The only issue priority
	  we will be tracking is whether or not the issue is "critical." If additional levels are needed
	  in the future, we will add them.
	- (If needed) Clean up the title to succinctly and clearly state the issue. Also ensure that
	  proposals are prefaced with "Proposal: [the rest of the title]".
	- Add the issue to the correct milestone. If any questions come up, don't worry about adding the
	  issue to a milestone until the questions are answered.
	- We attempt to do this process at least once per work day.
3. Discussion
	- Issues that are labeled `feature` or `proposal` must write a Nocalhost Improvement Proposal (NIP: Coming soon). Smaller quality-of-life enhancements are exempt.
	- Issues that are labeled as `feature` or `bug` should be connected to the PR that resolves it.
	- Whoever is working on a `feature` or `bug` issue (whether a maintainer or someone from the
	  community), should either assign the issue to themself or make a comment in the issue saying
	  that they are taking it.
	- `proposal` and `support/question` issues should stay open until resolved or if they have not
	  been active for more than 30 days. This will help keep the issue queue to a manageable size
	  and reduce noise. Should the issue need to stay open, the `keep open` label can be added.
4. Issue closure

## Pull Requests
We strongly welcome your pull request to make Nocalhost better.

### Branch Management
There are two main branches here:

1. `main` branch.
	1. It is the latest (pre-)release branch. We use `main` for tags, with version number `v0.4.10`, `v0.4.12` ...
	2. **Don't submit any PR on `main` branch.**
2. `dev` branch.
	1. It is our stable developing branch. After full testing, `dev` will be merged to `main` branch for the next release.
	2. **You are recommended to submit bugfix or feature PR on `dev` branch.**

Normal bugfix or feature request should be submitted to `dev` branch. After full testing, we will merge them to `main` branch for the next release.

### How to Contribute a Patch

1. Identify or create the related issue.
2. Fork the desired repo; develop and test your code changes.
3. Submit a pull request, making sure to sign your work and link the related issue.

Commit's conventions and standards are explained in the [commit specification
docs](https://github.com/nocalhost/nocalhost/blob/main/docs/contribute/commit-specification.md).

### Make Pull Requests
The code team will monitor all pull request, we run some code check and test on it. After all tests passed, we will accecpt this PR. But it won't merge to `main` branch at once, which have some delay.

Before submitting a pull request, please make sure the followings are done:

1. Fork the repo and create your branch from `main`.
2. Update code or documentation if you have changed APIs.
3. Add the copyright notice to the top of any new files you've added.
4. Check your code lints and checkstyles.
5. Test and test again your code.
6. Now, you can submit your pull request on `dev` branch.

### PR Lifecycle

1. PR creation
	- PRs are usually created to fix or else be a subset of other PRs that fix a particular issue.
	- We more than welcome PRs that are currently in progress. They are a great way to keep track of
	  important work that is in-flight, but useful for others to see. If a PR is a work in progress,
	  it **must** be prefaced with "WIP: [title]". Once the PR is ready for review, remove "WIP"
	  from the title.
	- It is preferred, but not required, to have a PR tied to a specific issue. There can be
	  circumstances where if it is a quick fix then an issue might be overkill. The details provided
	  in the PR description would suffice in this case.
2. Triage
	- The maintainer in charge of triaging will apply the proper labels for the issue. This should
	  include at least `bug` or `feature`, and `awaiting review` once all labels are
	  applied. See the [Labels section](#labels) for full details on the definitions of labels.
3. Assigning reviews
	- Once a review has the `awaiting review` label, maintainers will review them as schedule
	  permits. The maintainer who takes the issue should self-request a review.
4. Reviewing/Discussion
	- All reviews will be completed using GitHub review tool.
	- A "Comment" review should be used when there are questions about the code that should be
	  answered, but that don't involve code changes. This type of review does not count as approval.
	- A "Changes Requested" review indicates that changes to the code need to be made before they
	  will be merged.
	- Reviewers should update labels as needed (such as `needs rebase`)
5. Address comments by answering questions or changing code
6. LGTM (Looks good to me)
	- Once a Reviewer has completed a review and the code looks ready to merge, an "Approve" review
	  is used to signal to the contributor and to other maintainers that you have reviewed the code
	  and feel that it is ready to be merged.
7. Merge or close
	- PRs should stay open until merged or if they have not been active for more than 30 days. This
	  will help keep the PR queue to a manageable size and reduce noise. Should the PR need to stay
	  open (like in the case of a WIP), the `keep open` label can be added.
	  below to determine if
	  the PR requires more than one LGTM to merge.
	- If the owner of the PR is listed in the `OWNERS` file, that user **must** merge their own PRs
	  or explicitly request another OWNER do that for them.
	- If the owner of a PR is _not_ listed in `OWNERS`, any core maintainer may merge the PR.

#### Documentation PRs

Documentation PRs will follow the same lifecycle as other PRs. They will also be labeled with the
`docs` label. For documentation, special attention will be paid to spelling, grammar, and clarity
(whereas those things don't matter *as* much for comments in code).

## The Triager

Each month, one of the core maintainers will serve as the designated "triager" starting after the
public stand-up meetings on Thursday. This person will be in charge triaging new PRs and issues
throughout the work week.

## Labels

The following tables define all label types used for Nocalhost. It is split up by category.

### Common

| Label | Description |
| ----- | ----------- |
| `bug` | Marks an issue as a bug or a PR as a bugfix |
| `critical` | Marks an issue or PR as critical. This means that addressing the PR or issue is top priority and must be addressed as soon as possible |
| `docs` | Indicates the issue or PR is a documentation change |
| `feature` | Marks the issue as a feature request or a PR as a feature implementation |
| `keep open` | Denotes that the issue or PR should be kept open past 30 days of inactivity |
| `refactor` | Indicates that the issue is a code refactor and is not fixing a bug or adding additional functionality |

### Issue Specific

| Label | Description |
| ----- | ----------- |
| `help wanted` | Marks an issue needs help from the community to solve |
| `proposal` | Marks an issue as a proposal |
| `question/support` | Marks an issue as a support request or question |
| `good first issue` | Marks an issue as a good starter issue for someone new to Nocalhost |
| `wont fix` | Marks an issue as discussed and will not be implemented (or accepted in the case of a proposal) |

### PR Specific

| Label | Description |
| ----- | ----------- |
| `awaiting review` | Indicates a PR has been triaged and is ready for someone to review |
| `breaking` | Indicates a PR has breaking changes (such as API changes) |
| `in progress` | Indicates that a maintainer is looking at the PR, even if no review has been posted yet |
| `needs rebase` | Indicates a PR needs to be rebased before it can be merged |
| `needs pick` | Indicates a PR needs to be cherry-picked into a feature branch (generally bugfix branches). Once it has been, the `picked` label should be applied and this one removed |
| `picked` | This PR has been cherry-picked into a feature branch |

## License
By contributing to Nocalhost, you agree that your contributions will be licensed
under its [Apache 2.0 LICENSE](./LICENSE)