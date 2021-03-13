from kfp import dsl


class CELCondition(dsl.OpsGroup):
    def __init__(self, condition, name=None, cel_task=None):
        super(CELCondition, self).__init__('condition', name)
        self.condition = condition
        self.cel_task = cel_task


class CELTemplate():
    def __init__(self, name=None):
        self.name = name


    def template(self):
        base_template = {
            'name': "test-condition",
            'params': [{
                'name': "data",
                'value': "$(tasks.flip-coin.results.output) == heads"
            }],
            "taskRef": {
                "name": "cel_condition",
                "apiVersion": "cel.tekton.dev/v1alpha1",
                "kind": "CEL"
            }
        }
        return base_template


class CELParam(dsl.PipelineParam):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
