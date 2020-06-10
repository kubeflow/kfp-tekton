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

import os
import re

from setuptools import setup


NAME = "kfp-tekton"
# VERSION = .... Change the version in kfp_tekton/__init__.py
LICENSE = "Apache 2.0"
HOMEPAGE = "https://github.com/kubeflow/kfp-tekton/"

REQUIRES = [
    'kfp==0.5.1',
]


def find_version(*file_path_parts):
    here = os.path.abspath(os.path.dirname(__file__))
    with open(os.path.join(here, *file_path_parts), 'r') as fp:
        version_file_text = fp.read()

    version_match = re.search(
        r"^__version__ = ['\"]([^'\"]*)['\"]",
        version_file_text,
        re.M,
    )
    if version_match:
        return version_match.group(1)

    raise RuntimeError("Unable to find version string.")


setup(
    name=NAME,
    version=find_version("kfp_tekton", "__init__.py"),
    description="Kubeflow Pipelines DSL compiler generating Tekton YAML (instead of Argo YAML).",
    long_description="Extension of the Kubeflow Pipelines compiler generating Tekton YAML (instead of Argo YAML).",
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
