# Copyright 2020 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# =============================================================================
# To install the kfp_tekton project run:
#
#    $ pip3 install -e .
#
# To create a distribution for PyPi run:
#
#    $ export KFP_TEKTON_VERSION=0.4.0-rc1
#    $ python3 setup.py sdist
#    $ twine check dist/kfp-tekton-${KFP_TEKTON_VERSION/-rc/rc}.tar.gz
#    $ twine upload --repository pypi dist/kfp-tekton-${KFP_TEKTON_VERSION/-rc/rc}.tar.gz
#
#   ... or:
#
#    $ make distribution KFP_TEKTON_VERSION=0.4.0-rc1
#
# =============================================================================

import logging
import re
import sys

from os import environ as env
from os.path import abspath, dirname, join
from setuptools import setup


NAME = "kfp-tekton"
# VERSION = ... set the version in kfp_tekton/__init__.py or use KFP_TEKTON_VERSION env var
LICENSE = "Apache 2.0"
HOMEPAGE = "https://github.com/kubeflow/kfp-tekton/"
DESCRIPTION = "Tekton Compiler for Kubeflow Pipelines"
LONG_DESCRIPTION = """\
The kfp-tekton project is an extension of the Kubeflow Pipelines SDK compiler
generating Tekton YAML instead of Argo YAML. The project is still in an early
development stage. Contributions are welcome: {}
""".format(HOMEPAGE)

REQUIRES = [
    'kfp==1.0.4',
    'kubernetes==11.0.0'
]

logging.basicConfig()
logger = logging.getLogger("kfp_tekton/setup.py")
logger.setLevel(logging.INFO)


def find_version(*file_path_parts):

    if "KFP_TEKTON_VERSION" in env:
        version = env.get("KFP_TEKTON_VERSION")
        logger.warning("Using KFP_TEKTON_VERSION={}".format(version))
        return version

    here = abspath(dirname(__file__))
    with open(join(here, *file_path_parts), 'r') as fp:
        version_file_text = fp.read()

    version_match = re.search(
        r"^__version__ = ['\"]([^'\"]*)['\"]",
        version_file_text,
        re.M,
    )
    if version_match:
        return version_match.group(1)

    raise RuntimeError("Unable to find version string.")


def get_long_description() -> str:
    """Extract the content of the sdk/README.md and replace relative links when
    running `python3 setup.py sdist`. Return the abbreviated LONG_DESCRIPTION when
    running `pip install .`

    :return: long_description, Github flavored Markdown from the sdk/README.md
    """
    if "sdist" not in sys.argv:
        # log messages are only displayed when running `pip --verbose`
        logger.warning("This not a distribution build. Using abbreviated "
                       "long_description: \"{}\"".format(LONG_DESCRIPTION))
        return LONG_DESCRIPTION
    else:
        logger.warning(
            "Building a distribution. Using the sdk/README.md for the "
            "long_description to be displayed on the PyPi landing page.")

    here = abspath(dirname(__file__))
    project_root = join(here, "..", "..")
    sdk_readme_file = join(project_root, "sdk", "README.md")

    with open(sdk_readme_file, "r") as f:
        long_description = f.read()

    # replace relative links from sdk/README.md with be absolute links to GitHub
    github_repo_master_path = "{}/blob/master".format(HOMEPAGE.rstrip("/"))

    # replace relative links that are siblings to the README, i.e. [link text](FEATURES.md)
    long_description = re.sub(
        r"\[([^]]+)\]\((?!http|#|/)([^)]+)\)",
        r"[\1]({}/{}/\2)".format(github_repo_master_path, "sdk"),
        long_description)

    # replace links that are relative to the project root, i.e. [link text](/sdk/FEATURES.md)
    long_description = re.sub(
        r"\[([^]]+)\]\(/([^)]+)\)",
        r"[\1]({}/\2)".format(github_repo_master_path),
        long_description)

    # add anchor link targets around headings for links from table-of-contents (TOC)
    text = []
    for line in long_description.splitlines():
        matches = re.match(r'^(#+) (.*)$', line)
        if matches:
            line = '<h{}><a id="{}">{}</a></h{}>'.format(
                len(matches.group(1)),
                re.sub(r"[`.()]", "", matches.group(2).replace(" ", "-").lower()),
                matches.group(2),
                len(matches.group(1))
            )
        text.append(line)
    long_description = "\n".join(text)

    # generate the README with absolute links for easy side-by-side comparison
    sdk_readme_w_abs_links = join(project_root, "sdk", "README_for_PyPi.md")
    with open(sdk_readme_w_abs_links, 'w') as f:
        f.write(long_description)

    # verify all replaced links
    import requests  # top-level import fails pip install, only required for make dist
    invalid_links = []
    for (text, link) in re.findall(r"\[([^]]+)\]\((%s[^)]+)\)" % "http",
                                   long_description):
        logger.info("checking link: {}".format(link))
        response = requests.head(link, allow_redirects=True, timeout=3)
        if response.status_code >= 400:
            invalid_links.append((text, link))

    # report all invalid links
    if invalid_links:
        links = ["[{}]({})".format(text, link) for (text, link) in invalid_links]
        for l in links:
            logger.error("invalid link: {}".format(l))
        raise RuntimeError(
            "Invalid link(s) in long_description: {}"
            "Please open an issue at {}/issues".format(
                links, HOMEPAGE.rstrip("/")))

    return long_description


setup(
    name=NAME,
    version=find_version("kfp_tekton", "__init__.py"),
    description=DESCRIPTION,
    long_description=get_long_description(),
    long_description_content_type="text/markdown",
    author="kubeflow.org",
    license=LICENSE,
    url=HOMEPAGE,
    install_requires=REQUIRES,
    packages=[
        'kfp_tekton',
        'kfp_tekton.compiler',
    ],
    classifiers=[
        'Intended Audience :: Developers',
        'Intended Audience :: Education',
        'Intended Audience :: Science/Research',
        'License :: OSI Approved :: Apache Software License',
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.5',
        'Programming Language :: Python :: 3.6',
        'Programming Language :: Python :: 3.7',
        'Topic :: Scientific/Engineering',
        'Topic :: Scientific/Engineering :: Artificial Intelligence',
        'Topic :: Software Development',
        'Topic :: Software Development :: Libraries',
        'Topic :: Software Development :: Libraries :: Python Modules',
    ],
    python_requires='>=3.5.3',
    include_package_data=True,
    entry_points={
        'console_scripts': [
            'dsl-compile-tekton = kfp_tekton.compiler.main:main'
        ]
    })
