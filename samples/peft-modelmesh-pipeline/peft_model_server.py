from typing import List

from mlserver import MLModel, types
from mlserver.codecs import decode_args

from peft import PeftModel, PeftConfig
from transformers import AutoModelForCausalLM, AutoTokenizer
import torch
import os

class PeftModelServer(MLModel):
  async def load(self) -> bool:
      self._load_model()
      self.ready = True
      return self.ready

  @decode_args
  async def predict(self, content: List[str]) -> List[str]:
      return self._predict_outputs(content)

  def _load_model(self):
      model_name_or_path = os.environ.get("PRETRAINED_MODEL_PATH", "bigscience/bloomz-560m")
      peft_model_id = os.environ.get("PEFT_MODEL_ID", "aipipeline/bloomz-560m_PROMPT_TUNING_CAUSAL_LM")
      self.tokenizer = AutoTokenizer.from_pretrained(model_name_or_path)
      config = PeftConfig.from_pretrained(peft_model_id)
      self.model = AutoModelForCausalLM.from_pretrained(config.base_model_name_or_path)
      self.model = PeftModel.from_pretrained(self.model, peft_model_id)
      self.text_column = os.environ.get("DATASET_TEXT_COLUMN_NAME", "Tweet text")
      return

  def _predict_outputs(self, content: List[str]) -> List[str]:
      output_list = []
      for input in content:
        inputs = self.tokenizer(
            f'{self.text_column} : {input} Label : ',
            return_tensors="pt",
        )
        with torch.no_grad():
          inputs = {k: v for k, v in inputs.items()}
          outputs = self.model.generate(
              input_ids=inputs["input_ids"], attention_mask=inputs["attention_mask"], max_new_tokens=10, eos_token_id=3
          )
          outputs = self.tokenizer.batch_decode(outputs.detach().cpu().numpy(), skip_special_tokens=True)
        output_list.append(outputs[0])
      return output_list
