# TODO: choose appropriate base image, install Python, MLServer, and
# dependencies of your MLModel implementation
FROM python:3.8-slim-buster
RUN pip install mlserver peft transformers datasets
# ...

# The custom `MLModel` implementation should be on the Python search path
# instead of relying on the working directory of the image. If using a
# single-file module, this can be accomplished with:
COPY --chown=${USER} ./peft_model_server.py /opt/peft_model_server.py
ENV PYTHONPATH=/opt/

# environment variables to be compatible with ModelMesh Serving
# these can also be set in the ServingRuntime, but this is recommended for
# consistency when building and testing
ENV MLSERVER_MODELS_DIR=/models/_mlserver_models \
    MLSERVER_GRPC_PORT=8001 \
    MLSERVER_HTTP_PORT=8002 \
    MLSERVER_LOAD_MODELS_AT_STARTUP=false \
    MLSERVER_MODEL_NAME=peft-model

# With this setting, the implementation field is not required in the model
# settings which eases integration by allowing the built-in adapter to generate
# a basic model settings file
ENV MLSERVER_MODEL_IMPLEMENTATION=peft_model_server.PeftModelServer

CMD mlserver start ${MLSERVER_MODELS_DIR}