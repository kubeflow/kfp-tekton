# Copyright 2020 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import kfp.dsl as dsl
from kfp import components

COP_STR = """
name: cop
implementation:
  container:
    image: library/bash:4.4.23
    command:
    - sh
    - -c
    args:
    - echo foo > /mnt/file1
"""

cop_op = components.load_component_from_text(COP_STR)


@dsl.pipeline(
    name="volumeop-basic",
    description="A Basic Example on VolumeOp Usage."
)
def volumeop_basic(size: str = "10M"):
    vop = dsl.VolumeOp(
        name="create-pvc",
        resource_name="my-pvc",
        modes=dsl.VOLUME_MODE_RWO,
        size=size
        # success_condition="status.phase = Bound",
        # failure_condition="status.phase = Failed"
    )

    cop = cop_op().add_pvolumes({"/mnt": vop.volume})


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(volumeop_basic, __file__.replace('.py', '.yaml'))
