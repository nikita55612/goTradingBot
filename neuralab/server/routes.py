from aiohttp import web
import xgboost as xgb


async def ping(_: web.Request):
    return web.Response(text="pong")


async def predict(req: web.Request):
    res = {
        'data': {
            'predict': {},
            'error': '',
        },
        'status': 403,
    }
    if not req.body_exists:
        res['data']['error'] = 'request body is missing'
        return web.json_response(**res)

    try:
        body = await req.json()
    except ValueError:
        res['data']['error'] = 'invalid body format'
        return web.json_response(**res)

    for f in ('features', 'model'):
        if f not in body:
            res['data']['error'] = f'missing required field: "{f}"'
            return web.json_response(**res)

    features = body['features']
    model_name = body['model']

    if model_name not in req.app['models']:
        res['data']['error'] = f'no such model: "{model_name}"'
        return web.json_response(**res)

    model = req.app['models'][model_name]

    try:
        dmatrix = xgb.DMatrix(features)
        model_predict = model.predict(dmatrix).tolist()
        res['data']['predict'][model_name] = model_predict
    except Exception as e:
        res['data']['error'] = f'prediction failed: {str(e)}'
        res['status'] = 500
        return web.json_response(**res)

    res['status'] = 200
    return web.json_response(**res)
