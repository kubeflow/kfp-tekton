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

import sys
import os
from shutil import copyfile
from compiler.compiler_tests import TestTektonCompiler

if __name__ == '__main__':
    test_name = os.path.splitext(os.path.split(sys.argv[1])[1])[0]
    save_path = sys.argv[2]
    golden_path = os.path.join(os.path.dirname(__file__), 'compiler/testdata', test_name + '.yaml')

    if test_name == "compose":
        test = TestTektonCompiler().test_compose
    elif test_name == "basic_no_decorator":
        test = TestTektonCompiler().test_basic_no_decorator
    else:
        raise ValueError("Pipeline named '%s' is not recognized" % test_name)

    test()
    copyfile(golden_path, save_path)
