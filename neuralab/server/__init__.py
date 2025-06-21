from server import routes
from aiohttp import web
import xgboost as xgb
import os


def run(host: str, port: int):
	app = web.Application(client_max_size=4*1024*1024*1024)

	MODELS_PATH = "models"
	app['models'] = {}

	for model in os.listdir(MODELS_PATH):
		model_name = model.replace('.json', '')
		model_file = f'{MODELS_PATH}/{model}'
		app['models'][model_name] = xgb.Booster(model_file=model_file)

	app.add_routes(
        [
            web.get('/ping', routes.ping),
            web.post('/predict', routes.predict),
        ]
    )

	def on_start(_): return print(
        f'neuralab servis running on http://{host}:{port}')

	web.run_app(app, host=host, port=port, print=on_start)
