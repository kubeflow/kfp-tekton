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

import re

from kfp import dsl


def sanitize_k8s_name(name,
                      allow_capital_underscore=False,
                      allow_dot=False,
                      allow_slash=False,
                      max_length=63,
                      suffix_space=0):
    """From _make_kubernetes_name
      sanitize_k8s_name cleans and converts the names in the workflow.

    https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
    https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
    https://github.com/kubernetes/kubernetes/blob/c369cf18/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go#L89

    Args:
      name: original name
      allow_capital_underscore: whether to allow capital letter and underscore
        in this name (i.e. for parameters)
      allow_dot: whether to allow dots in this name (i.e. for labels)
      allow_slash: whether to allow slash in this name (i.e. for label and annotation keys)
      max_length: maximum length of K8s name, default: 63
      suffix_space: number of characters reserved for a suffix to be appended

    Returns:
      sanitized name.
    """
    k8s_name = re.sub('[^-_./0-9A-Za-z]+', '-', name)

    if not allow_capital_underscore:
      k8s_name = re.sub('_', '-', k8s_name.lower())

    if not allow_dot:
      k8s_name = re.sub('[.]', '-', k8s_name)

    if not allow_slash:
      k8s_name = re.sub('[/]', '-', k8s_name)

    # replace duplicate dashes, strip enclosing dashes
    k8s_name = re.sub('-+', '-', k8s_name).strip('-')

    # truncate if length exceeds max_length
    max_length = max_length - suffix_space
    k8s_name = k8s_name[:max_length].rstrip('-') if len(k8s_name) > max_length else k8s_name

    return k8s_name


def convert_k8s_obj_to_json(k8s_obj):
    """
    Builds a JSON K8s object.

    If obj is None, return None.
    If obj is str, int, long, float, bool, return directly.
    If obj is datetime.datetime, datetime.date
        convert to string in iso8601 format.
    If obj is list, sanitize each element in the list.
    If obj is dict, return the dict.
    If obj is swagger model, return the properties dict.

    Args:
      obj: The data to serialize.
    Returns: The serialized form of data.
    """

    from six import text_type, integer_types, iteritems
    PRIMITIVE_TYPES = (float, bool, bytes, text_type) + integer_types
    from datetime import date, datetime

    if k8s_obj is None:
      return None
    elif isinstance(k8s_obj, PRIMITIVE_TYPES):
      return k8s_obj
    elif isinstance(k8s_obj, list):
      return [convert_k8s_obj_to_json(sub_obj)
              for sub_obj in k8s_obj]
    elif isinstance(k8s_obj, tuple):
      return tuple(convert_k8s_obj_to_json(sub_obj)
                   for sub_obj in k8s_obj)
    elif isinstance(k8s_obj, (datetime, date)):
      return k8s_obj.isoformat()
    elif isinstance(k8s_obj, dsl.PipelineParam):
      if isinstance(k8s_obj.value, str):
        return k8s_obj.value
      return '$(inputs.params.%s)' % k8s_obj.full_name  # change for Tekton
    
    if isinstance(k8s_obj, dict):
      obj_dict = k8s_obj
    else:
      # Convert model obj to dict except
      # attributes `swagger_types`, `attribute_map`
      # and attributes which value is not None.
      # Convert attribute name to json key in
      # model definition for request.
      obj_dict = {k8s_obj.attribute_map[attr]: getattr(k8s_obj, attr)
                  for attr in k8s_obj.attribute_map
                  if getattr(k8s_obj, attr) is not None}

    return {key: convert_k8s_obj_to_json(val)
            for key, val in iteritems(obj_dict)}


def _to_bool(b):
    if type(b) == str:
      b = b.lower()
      if b != 'true' and b != 'false':
        raise ValueError(
          'Invalid boolean string {}. Should be "true" or "false".'.format(b))
      else:
        b = b == 'true'
    elif type(b) != bool:
      raise ValueError('Invalid value {}. Should be boolean.'.format(b))
    return b


def _to_int(i):
    try:
      result = int(i)
    except ValueError:
      raise ValueError('Invalid {}. Should be integer.'.format(i))
    return result


def _to_float(f):
    try:
      result = float(f)
    except ValueError:
      raise ValueError('Invalid {}. Should be float.'.format(f))
    return result


def sanitize_k8s_object(k8s_obj, type=None):
    """
    Recursively sanitize and cast k8s object type based on the type definition
    in kubernetes.client.models.

    Args:
      k8s_obj: k8s object
    """
    from six import text_type, integer_types
    PRIMITIVE_TYPES = (float, bool, bytes, text_type) + integer_types
    from datetime import date, datetime

    if k8s_obj is None:
      return None
    elif isinstance(k8s_obj, PRIMITIVE_TYPES):
      if type == 'str':
        return str(k8s_obj)
      elif type == 'int':
        return _to_int(k8s_obj)
      elif type == 'float':
        return _to_float(k8s_obj)
      elif type == 'bool':
        return _to_bool(k8s_obj)
      else:
        return k8s_obj
    elif isinstance(k8s_obj, list):
      if type == 'list[str]':
        return [sanitize_k8s_object(sub_obj, 'str')
                for sub_obj in k8s_obj]
      else:
        return [sanitize_k8s_object(sub_obj, None)
                for sub_obj in k8s_obj]
    elif isinstance(k8s_obj, tuple):
      if type == 'list[str]':
        return tuple(sanitize_k8s_object(sub_obj, 'str')
                    for sub_obj in k8s_obj)
      else:
        return tuple(sanitize_k8s_object(sub_obj, None)
                    for sub_obj in k8s_obj)
    elif isinstance(k8s_obj, (datetime, date)):
      return k8s_obj
    
    if isinstance(k8s_obj, dict):
      return k8s_obj
    else:
      for attr in k8s_obj.attribute_map:
        if getattr(k8s_obj, attr) is not None:
          type = k8s_obj.openapi_types[attr]
          value = getattr(k8s_obj, attr)
          value = sanitize_k8s_object(value, type)
          setattr(k8s_obj, attr, value)
      return k8s_obj
