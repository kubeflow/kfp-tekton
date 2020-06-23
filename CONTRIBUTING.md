<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [How to Contribute](#how-to-contribute)
  - [Contributor License Agreement](#contributor-license-agreement)
  - [Development Guidelines](#development-guidelines)
  - [Code reviews](#code-reviews)
  - [Get involved](#get-involved)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# How to Contribute

We'd love to accept your patches and contributions to this project. There are
just a few small guidelines you need to follow.

## Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License
Agreement. You (or your employer) retain the copyright to your contribution,
this simply gives us permission to use and redistribute your contributions as
part of the project. Head over to <https://cla.developers.google.com/> to see
your current agreements on file or to sign a new one.

You generally only need to submit a CLA once, so if you've already submitted one
(even if it was for a different project), you probably don't need to do it
again.

## Development Guidelines

Please take a look at the [KFP-Tekton Developer Guide](sdk/python/README.md) for
details about how to make code contributions to the KFP-Tekton project.

## Coding style

The Python part of the project will follow [Google Python style guide](http://google.github.io/styleguide/pyguide.html).
We provide a [yapf](https://github.com/google/yapf) configuration file to help
contributors auto-format their code to adopt the Google Python style. Also, it
is encouraged to lint python docstrings by [docformatter](https://github.com/myint/docformatter).

The frontend part of the project uses [prettier](https://prettier.io/) for
formatting, read [frontend/README.md#code-style](frontend/README.md#code-style)
for more details.

## Code reviews

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult
[GitHub Help](https://help.github.com/articles/about-pull-requests/) for more
information on using pull requests.

The following should be viewed as Best Practices unless you know better ones
(please submit a guidelines PR).

| Practice | Rationale |
| -------- | --------- |
| Keep the code clean | The health of the codebase is imperative to the success of the project. Files should be under 500 lines long in most cases, which may mean a refactor is necessary before adding changes. |
| Limit your scope | No one wants to review a 1000 line PR. Try to keep your changes focused to ease reviewability. This may mean separating a large feature into several smaller milestones.  |
| Refine commit messages | Your commit messages should be in the imperative tense and clearly describe your feature upon first glance. See [this article](https://chris.beams.io/posts/git-commit/) for guidelines.
| Reference an issue | Issues are a great way to gather design feedback from the community. To save yourself time on a controversial PR, first cut an issue for any major feature work. |

## Get involved

* [Slack](http://kubeflow.slack.com/)
* [Twitter](http://twitter.com/kubeflow)
* [Mailing List](https://groups.google.com/forum/#!forum/kubeflow-discuss)
