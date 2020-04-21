#!/usr/bin/env python3
# Copyright 2019 Google LLC
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

import kfp
from kfp import dsl

def print_op(msg):
    """Print a message."""
    return dsl.ContainerOp(
        name='Print',
        image='alpine:3.6',
        command=['echo', msg],
    )
    

@dsl.pipeline(
    name='Conditional Example Pipeline',
    description='Shows how to use dsl.Condition().'
)
def conditional_pipeline(num: int = 5):

    with dsl.Condition(num == 5):
        print_op('Number is equal to 5')
    with dsl.Condition(num != 5):
        print_op('Number is not equal to 5')


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(conditional_pipeline, __file__ + '.yaml')
