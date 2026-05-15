import grpc
from concurrent import futures
import time
import pandas as pd
from prophet import Prophet
import predictor_pb2
import predictor_pb2_grpc
import os

class PredictorServicer(predictor_pb2_grpc.PredictorServicer):
    def Predict(self, request, context):
        print(f"Received prediction request with {len(request.data_points)} data points")
        
        if len(request.data_points) < 2:
            print("Not enough data points for prediction")
            return predictor_pb2.PredictionResponse(predicted_replicas=1)

        data = []
        for dp in request.data_points:
            data.append({
                'ds': pd.to_datetime(dp.timestamp, unit='s'),
                'y': dp.value
            })
        
        df = pd.DataFrame(data)
        
        try:
            m = Prophet(yearly_seasonality=False, weekly_seasonality=False, daily_seasonality=True)
            m.fit(df)
            
            future = m.make_future_dataframe(periods=1, freq=f'{request.forecast_horizon_seconds}s', include_history=False)
            
            forecast = m.predict(future)
            
            predicted_value = forecast['yhat'].iloc[0]
            
            predicted_replicas = int(round(max(1, predicted_value)))
            
            print(f"Predicted replicas: {predicted_replicas} (raw value: {predicted_value:.2f})")
            return predictor_pb2.PredictionResponse(predicted_replicas=predicted_replicas)
            
        except Exception as e:
            print(f"Error during prediction: {e}")
            return predictor_pb2.PredictionResponse(predicted_replicas=1)

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    predictor_pb2_grpc.add_PredictorServicer_to_server(PredictorServicer(), server)
    port = os.environ.get("PORT", "50051")
    server.add_insecure_port(f'[::]:{port}')
    print(f"Starting gRPC server on port {port}...")
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
