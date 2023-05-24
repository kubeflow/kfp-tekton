# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


from typing import Optional

from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.container_component
def training_configurator_and_validator(
    dataset_stats: Input[Artifact],
    split_example_counts: str,
    training_schema: Input[Artifact],
    instance_schema: Input[Artifact],
    metadata: Output[Artifact],
    instance_baseline: Output[Artifact],
    target_column: Optional[str] = '',
    weight_column: Optional[str] = '',
    prediction_type: Optional[str] = '',
    optimization_objective: Optional[str] = '',
    optimization_objective_recall_value: Optional[float] = -1,
    optimization_objective_precision_value: Optional[float] = -1,
    run_evaluation: Optional[bool] = False,
    run_distill: Optional[bool] = False,
    enable_probabilistic_inference: Optional[bool] = False,
    time_series_identifier_column: Optional[str] = '',
    time_column: Optional[str] = '',
    time_series_attribute_columns: Optional[str] = '',
    available_at_forecast_columns: Optional[str] = '',
    unavailable_at_forecast_columns: Optional[str] = '',
    quantiles: Optional[str] = '',
    context_window: Optional[int] = -1,
    forecast_horizon: Optional[int] = -1,
    forecasting_model_type: Optional[str] = '',
    forecasting_transformations_path: Optional[str] = '',
    stage_1_deadline_hours: Optional[float] = None,
    stage_2_deadline_hours: Optional[float] = None,
):
  # fmt: off
  """Component to configure training and validate data and user-input
  configurations.

  Args:
      dataset_stats: Dataset stats generated by
        feature transform engine.
      split_example_counts: JSON string of data split example counts for
        train, validate, and test splits.
      training_schema_path: Schema of input data to the tf_model
        at training time.
      instance_schema: Schema of input data to the tf_model at
        serving time.
      target_column: Target column of input data.
      weight_column: Weight column of input data.
      prediction_type: Model prediction type. One of "classification",
        "regression", "time_series".
      optimization_objective: Objective function the model is optimizing
        towards. The training process creates a model that maximizes/minimizes
        the value of the objective function over the validation set. The
        supported optimization objectives depend on the prediction type. If the
        field is not set, a default objective function is used.
          classification: "maximize-au-roc" (default) - Maximize the
            area under the receiver operating characteristic (ROC) curve.
            "minimize-log-loss" - Minimize log loss. "maximize-au-prc" -
            Maximize the area under the precision-recall curve.
            "maximize-precision-at-recall" - Maximize precision for a specified
            recall value. "maximize-recall-at-precision" - Maximize recall for a
            specified precision value.
          classification (multi-class): "minimize-log-loss" (default) - Minimize
            log loss.
          regression: "minimize-rmse" (default) - Minimize root-mean-squared
            error (RMSE). "minimize-mae" - Minimize mean-absolute error (MAE).
            "minimize-rmsle" - Minimize root-mean-squared log error (RMSLE).
      optimization_objective_recall_value: Required when
        optimization_objective is "maximize-precision-at-recall". Must be
        between 0 and 1, inclusive.
      optimization_objective_precision_value: Required when
        optimization_objective is "maximize-recall-at-precision". Must be
        between 0 and 1, inclusive.
      run_evaluation: Whether we are running evaluation in the training
        pipeline.
      run_distill: Whether the distillation should be applied to the
        training.
      enable_probabilistic_inference: If probabilistic inference is
        enabled, the model will fit a distribution that captures the uncertainty
        of a prediction. At inference time, the predictive distribution is used
        to make a point prediction that minimizes the optimization objective.
        For example, the mean of a predictive distribution is the point
        prediction that minimizes RMSE loss. If quantiles are specified, then
        the quantiles of the distribution are also returned.
      time_series_identifier_column: Time series idenfier column. Used by
        forecasting only.
      time_column: The column that indicates the time. Used by forecasting
        only.
      time_series_attribute_columns: The column names of the time series
        attributes.
      available_at_forecast_columns: The names of the columns that are
        available at forecast time.
      unavailable_at_forecast_columns: The names of the columns that are
        not available at forecast time.
      quantiles: All quantiles that the model need to predict.
      context_window: The length of the context window.
      forecast_horizon: The length of the forecast horizon.
      forecasting_model_type: The model types, e.g. l2l, seq2seq, tft.
      forecasting_transformations_path: The path to the JSON format
        forecasting transformations. Used by forecasting only.
      stage_1_deadline_hours: Stage 1 training budget in
        hours.
      stage_2_deadline_hours: Stage 2 training budget in
        hours.

  Returns:
      metadata: The tabular example gen metadata.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image='us-docker.pkg.dev/vertex-ai/automl-tabular/feature-transform-engine:20230424_1325',
      command=[],
      args=[
          'training_configurator_and_validator',
          dsl.ConcatPlaceholder(
              items=['--instance_schema_path=', instance_schema.uri]
          ),
          dsl.ConcatPlaceholder(
              items=['--training_schema_path=', training_schema.uri]
          ),
          dsl.ConcatPlaceholder(
              items=['--dataset_stats_path=', dataset_stats.uri]
          ),
          dsl.ConcatPlaceholder(
              items=['--split_example_counts=', split_example_counts]
          ),
          dsl.ConcatPlaceholder(items=['--target_column=', target_column]),
          dsl.ConcatPlaceholder(items=['--weight_column=', weight_column]),
          dsl.ConcatPlaceholder(items=['--prediction_type=', prediction_type]),
          dsl.ConcatPlaceholder(
              items=['--optimization_objective=', optimization_objective]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--optimization_objective_recall_value=',
                  optimization_objective_recall_value,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--optimization_objective_precision_value=',
                  optimization_objective_precision_value,
              ]
          ),
          dsl.ConcatPlaceholder(items=['--metadata_path=', metadata.uri]),
          dsl.ConcatPlaceholder(
              items=['--instance_baseline_path=', instance_baseline.uri]
          ),
          dsl.ConcatPlaceholder(items=['--run_evaluation=', run_evaluation]),
          dsl.ConcatPlaceholder(items=['--run_distill=', run_distill]),
          dsl.ConcatPlaceholder(
              items=[
                  '--enable_probabilistic_inference=',
                  enable_probabilistic_inference,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--time_series_identifier_column=',
                  time_series_identifier_column,
              ]
          ),
          dsl.ConcatPlaceholder(items=['--time_column=', time_column]),
          dsl.ConcatPlaceholder(
              items=[
                  '--time_series_attribute_columns=',
                  time_series_attribute_columns,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--available_at_forecast_columns=',
                  available_at_forecast_columns,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--unavailable_at_forecast_columns=',
                  unavailable_at_forecast_columns,
              ]
          ),
          dsl.ConcatPlaceholder(items=['--quantiles=', quantiles]),
          dsl.ConcatPlaceholder(items=['--context_window=', context_window]),
          dsl.ConcatPlaceholder(
              items=['--forecast_horizon=', forecast_horizon]
          ),
          dsl.ConcatPlaceholder(
              items=['--forecasting_model_type=', forecasting_model_type]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--forecasting_transformations_path=',
                  forecasting_transformations_path,
              ]
          ),
          dsl.IfPresentPlaceholder(
              input_name='stage_1_deadline_hours',
              then=dsl.ConcatPlaceholder(
                  items=[
                      '--stage_1_deadline_hours=',
                      stage_1_deadline_hours,
                  ]
              ),
          ),
          dsl.IfPresentPlaceholder(
              input_name='stage_2_deadline_hours',
              then=dsl.ConcatPlaceholder(
                  items=[
                      '--stage_2_deadline_hours=',
                      stage_2_deadline_hours,
                  ]
              ),
          ),
      ],
  )
