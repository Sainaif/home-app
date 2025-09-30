"""
ML Forecasting Service for Holy Home
Provides time-series forecasting using SARIMAX, Holt-Winters, and Simple ES
"""
import logging
import sys
from datetime import datetime
from typing import Optional

import uvicorn
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

from app.models import ForecastRequest, ForecastResponse, ModelType
from app.forecaster import Forecaster

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='{"ts":"%(asctime)s","level":"%(levelname)s","service":"ml","message":"%(message)s"}',
    stream=sys.stdout
)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="Holy Home ML Service",
    description="Time-series forecasting for household utilities",
    version="1.0.0"
)

# Initialize forecaster
forecaster = Forecaster()


@app.get("/healthz")
async def healthcheck():
    """Health check endpoint"""
    return {
        "status": "ok",
        "service": "ml",
        "time": datetime.now().isoformat()
    }


@app.post("/forecast", response_model=ForecastResponse)
async def forecast(request: ForecastRequest) -> ForecastResponse:
    """
    Generate forecast based on historical time series data

    Model selection:
    - ≥24 points: SARIMAX with seasonal period 12
    - 12-23 points: Holt-Winters ExponentialSmoothing
    - <12 points: Simple Exponential Smoothing

    Returns predictions with confidence intervals
    """
    try:
        logger.info(f"Forecast request received: target={request.target}, "
                   f"series_length={len(request.historical_values)}, "
                   f"horizon={request.horizon_months}")

        result = forecaster.forecast(request)

        logger.info(f"Forecast completed: model={result.model.name}, "
                   f"predicted_values={len(result.predicted_values)}")

        return result

    except ValueError as e:
        logger.error(f"Validation error: {str(e)}")
        raise HTTPException(status_code=400, detail=str(e))

    except Exception as e:
        logger.error(f"Forecasting error: {str(e)}", exc_info=True)
        raise HTTPException(status_code=500, detail="Internal forecasting error")


if __name__ == "__main__":
    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=8000,
        log_config=None  # Use our custom logging
    )