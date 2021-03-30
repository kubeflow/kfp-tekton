# Copyright 2019-2020 kubeflow.org.
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

from collections import OrderedDict
from typing import Any, Union, Optional, TextIO, overload
import yaml

__all__ = [
    "dump_yaml"
]


class _Dumper(yaml.SafeDumper):
    def ignore_aliases(self, *args, **kwargs):
        return True


def _dict_representer(dumper, data):
    return dumper.represent_mapping(
        yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG,
        data.items()
    )


_Dumper.add_representer(OrderedDict, _dict_representer)
_Dumper.add_representer(dict, _dict_representer)


# Hack to force the code (multi-line string) to be output using the '|' style.
def represent_str_or_text(self, data):
    style = None
    if data.find('\n') >= 0:  # Multiple lines
        style = '|'
    # Fix for bool-like strings
    if data.lower() in ['y', 'n', 'yes', 'no', 'true', 'false', 'on', 'off']:
        style = '"'
    return self.represent_scalar(u'tag:yaml.org,2002:str', data, style)


_Dumper.add_representer(str, represent_str_or_text)


@overload
def dump_yaml(data: Any) -> Union[str, bytes]: ...
@overload
def dump_yaml(data: Any, stream: None) -> Union[str, bytes]: ...
@overload
def dump_yaml(data: Any, stream: TextIO) -> type(None): ...


def dump_yaml(data: Any, stream: Optional[TextIO] = None, **kwargs) -> Optional[Union[str, bytes]]:
    # PyYAML doesn't handle bool-like strings properly, so it needs to be fixed.
    # See https://github.com/yaml/pyyaml/issues/247

    return yaml.dump(data, stream, _Dumper, default_flow_style=False, **kwargs)

