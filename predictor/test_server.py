import pytest
import predictor_pb2
from server import PredictorServicer
import time

def test_predict_not_enough_data():
    servicer = PredictorServicer()
    request = predictor_pb2.PredictionRequest(
        data_points=[
            predictor_pb2.DataPoint(timestamp=int(time.time()), value=1.0)
        ],
        forecast_horizon_seconds=60
    )
    response = servicer.Predict(request, None)
    assert response.predicted_replicas == 1

def test_predict_success():
    servicer = PredictorServicer()
    now = int(time.time())
    request = predictor_pb2.PredictionRequest(
        data_points=[
            predictor_pb2.DataPoint(timestamp=now - 300, value=1.0),
            predictor_pb2.DataPoint(timestamp=now - 240, value=1.1),
            predictor_pb2.DataPoint(timestamp=now - 180, value=1.2),
            predictor_pb2.DataPoint(timestamp=now - 120, value=1.3),
            predictor_pb2.DataPoint(timestamp=now - 60, value=1.4),
            predictor_pb2.DataPoint(timestamp=now, value=1.5),
        ],
        forecast_horizon_seconds=60
    )
    response = servicer.Predict(request, None)
    assert response.predicted_replicas >= 1
