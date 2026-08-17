import traceback
from tokenizers import Tokenizer
from nuxt import route, logger, Request
from nuxt.repositorys.validation import fields, use_args

tokenizer = Tokenizer.from_file("./tokenizer.json")

support_models = set([
    "deepseek-v4-pro",
    "deepseek-chat",
    "deepseek-coder"
])

def num_tokens_from_messages(messages, model="deepseek-v4-pro"):

    if model not in support_models:
        raise NotImplementedError(
            f"num_tokens_from_messages() is not presently implemented for model {model}."
        )
    num_tokens = 0
    for message in messages:
        num_tokens += 4  
        for key, value in message.items():
            if value:
                num_tokens += len(tokenizer.encode(value, add_special_tokens=False).ids)
            if key == "name":
                num_tokens += -1  

    num_tokens += 3  
    return num_tokens

@route("/tokenizer/<str:model_name>", methods=["POST"])
@use_args({
    "role": fields.Str(required=True),
    "content": fields.Str(missing=""),
    "name": fields.Str(required=False)
}, location="json")
def get_num_tokens(req: Request, message: dict, model_name: str):
    try:
        return {
            "code": 200,
            "num_tokens": num_tokens_from_messages([message], model=model_name)
        }
    except Exception as e:
        logger.error(traceback.format_exc())
        return {
            "code": 500,
            "msg": f"{e}"
        }
