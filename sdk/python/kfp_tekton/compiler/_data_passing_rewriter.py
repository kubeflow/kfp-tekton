# Copyright 2019-2020 kubeflow.org
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

# from typing import List, Text, Dict, Any


def fix_big_data_passing(workflow: dict) -> dict:
    """
    Currently this function does not do anything.

    TODO: evaluate if an implementation similar to the upstream function in KFP
          is required

    Documentation from KFP:

    fix_big_data_passing converts a workflow where some artifact data is passed
    as parameters and converts it to a workflow where this data is passed as
    artifacts.

    Args:
        workflow: The workflow to fix
    Returns:
        The fixed workflow

    Motivation:

    DSL compiler only supports passing Argo parameters.
    Due to the convoluted nature of the DSL compiler, the artifact consumption
    and passing has been implemented on top of that using parameter passing.
    The artifact data is passed as parameters and the consumer template creates
    an artifact/file out of that data.
    Due to the limitations of Kubernetes and Argo this scheme cannot pass data
    larger than few kilobytes preventing any serious use of artifacts.

    This function rewrites the compiled workflow so that the data consumed as
    artifact is passed as artifact.
    It also prunes the unused parameter outputs. This is important since if a
    big piece of data is ever returned through a file that is also output as
    parameter, the execution will fail.
    This makes is possible to pass large amounts of data.

    Implementation:

    1. Index the DAGs to understand how data is being passed and which inputs/outputs
       are connected to each other.
    2. Search for direct data consumers in container/resource templates and some
       DAG task attributes (e.g. conditions and loops) to find out which inputs
       are directly consumed as parameters/artifacts.
    3. Propagate the consumption information upstream to all inputs/outputs all
       the way up to the data producers.
    4. Convert the inputs, outputs and arguments based on how they're consumed
       downstream.
    """
    return workflow
