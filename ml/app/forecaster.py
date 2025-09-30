"""
Time-series forecasting engine
Supports SARIMAX, Holt-Winters, and Simple Exponential Smoothing
"""
import logging
from datetime import datetime, timedelta
from typing import List, Tuple

import numpy as np
import pandas as pd
from statsmodels.tsa.exponential_smoothing.ets import ETSModel
from statsmodels.tsa.holtwinters import ExponentialSmoothing
from statsmodels.tsa.statespace.sarimax import SARIMAX

from app.models import (
    ForecastRequest,
    ForecastResponse,
    ModelInfo,
    ModelType,
    ConfidenceInterval
)

logger = logging.getLogger(__name__)


class Forecaster:
    """Time-series forecaster with automatic model selection"""

    def forecast(self, request: ForecastRequest) -> ForecastResponse:
        """Generate forecast using appropriate model based on series length"""
        # Validate input
        if len(request.historical_values) < 3:
            raise ValueError("Need at least 3 historical data points for forecasting")

        if len(request.historical_dates) != len(request.historical_values):
            raise ValueError("historical_dates and historical_values must have same length")

        # Prepare data
        series = pd.Series(
            request.historical_values,
            index=pd.to_datetime(request.historical_dates)
        )

        # Select model based on series length
        n = len(series)
        logger.info(f"Series length: {n}, selecting model...")

        if n >= 24:
            model_result = self._fit_sarimax(series, request.horizon_months, request.confidence_level)
        elif n >= 12:
            model_result = self._fit_holt_winters(series, request.horizon_months, request.confidence_level)
        else:
            model_result = self._fit_simple_es(series, request.horizon_months, request.confidence_level)

        predictions, lower, upper, model_info = model_result

        # Generate forecast dates
        last_date = series.index[-1]
        forecast_dates = [
            (last_date + timedelta(days=30 * i)).isoformat()
            for i in range(1, request.horizon_months + 1)
        ]

        # Calculate costs if cost_per_unit provided
        predicted_costs = None
        if request.cost_per_unit is not None:
            predicted_costs = [v * request.cost_per_unit for v in predictions]

        return ForecastResponse(
            target=request.target,
            model=model_info,
            predicted_dates=forecast_dates,
            predicted_values=predictions,
            confidence_interval=ConfidenceInterval(lower=lower, upper=upper),
            predicted_costs=predicted_costs
        )

    def _fit_sarimax(
        self,
        series: pd.Series,
        horizon: int,
        confidence: float
    ) -> Tuple[List[float], List[float], List[float], ModelInfo]:
        """Fit SARIMAX model with seasonal period 12"""
        logger.info("Fitting SARIMAX model...")

        # Grid search for best parameters (simplified)
        best_aic = float('inf')
        best_model = None
        best_params = None

        for p in [1, 2]:
            for d in [0, 1]:
                for q in [1, 2]:
                    try:
                        model = SARIMAX(
                            series,
                            order=(p, d, q),
                            seasonal_order=(1, 0, 1, 12),
                            enforce_stationarity=False,
                            enforce_invertibility=False
                        )
                        fitted = model.fit(disp=False, maxiter=50)

                        if fitted.aic < best_aic:
                            best_aic = fitted.aic
                            best_model = fitted
                            best_params = (p, d, q)

                    except Exception as e:
                        logger.debug(f"SARIMAX fit failed for ({p},{d},{q}): {e}")
                        continue

        if best_model is None:
            logger.warning("SARIMAX fit failed, falling back to Holt-Winters")
            return self._fit_holt_winters(series, horizon, confidence)

        logger.info(f"Best SARIMAX params: {best_params}, AIC: {best_aic:.2f}")

        # Generate forecast
        forecast_result = best_model.get_forecast(steps=horizon)
        predictions = forecast_result.predicted_mean.tolist()

        # Confidence intervals
        conf_int = forecast_result.conf_int(alpha=1 - confidence)
        lower = conf_int.iloc[:, 0].tolist()
        upper = conf_int.iloc[:, 1].tolist()

        # Ensure non-negative predictions
        predictions = [max(0, v) for v in predictions]
        lower = [max(0, v) for v in lower]
        upper = [max(0, v) for v in upper]

        model_info = ModelInfo(
            name=ModelType.SARIMAX,
            parameters={
                "order": best_params,
                "seasonal_order": (1, 0, 1, 12)
            },
            fit_stats={
                "aic": float(best_aic),
                "series_length": len(series)
            }
        )

        return predictions, lower, upper, model_info

    def _fit_holt_winters(
        self,
        series: pd.Series,
        horizon: int,
        confidence: float
    ) -> Tuple[List[float], List[float], List[float], ModelInfo]:
        """Fit Holt-Winters ExponentialSmoothing"""
        logger.info("Fitting Holt-Winters model...")

        try:
            # Use additive trend and seasonality
            model = ExponentialSmoothing(
                series,
                trend='add',
                seasonal='add',
                seasonal_periods=min(12, len(series) // 2)
            )
            fitted = model.fit()

            # Generate forecast
            forecast = fitted.forecast(steps=horizon)
            predictions = forecast.tolist()

            # Estimate confidence intervals using residuals
            residuals = fitted.fittedvalues - series
            std_error = np.std(residuals)

            # Simple confidence interval estimation
            z_score = 1.96 if confidence >= 0.95 else 1.645
            lower = [max(0, p - z_score * std_error * np.sqrt(i + 1)) for i, p in enumerate(predictions)]
            upper = [p + z_score * std_error * np.sqrt(i + 1) for i, p in enumerate(predictions)]

            # Ensure non-negative
            predictions = [max(0, v) for v in predictions]

            model_info = ModelInfo(
                name=ModelType.HOLT_WINTERS,
                parameters={
                    "trend": "add",
                    "seasonal": "add",
                    "seasonal_periods": min(12, len(series) // 2)
                },
                fit_stats={
                    "series_length": len(series),
                    "residual_std": float(std_error)
                }
            )

            return predictions, lower, upper, model_info

        except Exception as e:
            logger.warning(f"Holt-Winters fit failed: {e}, falling back to Simple ES")
            return self._fit_simple_es(series, horizon, confidence)

    def _fit_simple_es(
        self,
        series: pd.Series,
        horizon: int,
        confidence: float
    ) -> Tuple[List[float], List[float], List[float], ModelInfo]:
        """Fit Simple Exponential Smoothing or Moving Average"""
        logger.info("Fitting Simple Exponential Smoothing...")

        try:
            model = ExponentialSmoothing(series, trend=None, seasonal=None)
            fitted = model.fit()

            # Generate forecast
            forecast = fitted.forecast(steps=horizon)
            predictions = forecast.tolist()

            # Estimate confidence intervals
            residuals = fitted.fittedvalues - series
            std_error = np.std(residuals)

            z_score = 1.96 if confidence >= 0.95 else 1.645
            lower = [max(0, p - z_score * std_error * np.sqrt(i + 1)) for i, p in enumerate(predictions)]
            upper = [p + z_score * std_error * np.sqrt(i + 1) for i, p in enumerate(predictions)]

            # Ensure non-negative
            predictions = [max(0, v) for v in predictions]

            model_info = ModelInfo(
                name=ModelType.SIMPLE_ES,
                parameters={"smoothing_level": float(fitted.params['smoothing_level'])},
                fit_stats={
                    "series_length": len(series),
                    "residual_std": float(std_error)
                }
            )

            return predictions, lower, upper, model_info

        except Exception as e:
            logger.warning(f"Simple ES fit failed: {e}, using moving average")
            return self._fit_moving_average(series, horizon, confidence)

    def _fit_moving_average(
        self,
        series: pd.Series,
        horizon: int,
        confidence: float
    ) -> Tuple[List[float], List[float], List[float], ModelInfo]:
        """Fallback: simple moving average"""
        logger.info("Using Moving Average as fallback...")

        window = min(3, len(series))
        ma = series.rolling(window=window).mean().iloc[-1]

        # Naive forecast: repeat last moving average
        predictions = [float(ma)] * horizon

        # Simple confidence interval based on historical variance
        std = series.std()
        z_score = 1.96 if confidence >= 0.95 else 1.645

        lower = [max(0, ma - z_score * std)] * horizon
        upper = [ma + z_score * std] * horizon

        model_info = ModelInfo(
            name=ModelType.MOVING_AVERAGE,
            parameters={"window": window},
            fit_stats={
                "series_length": len(series),
                "mean": float(ma),
                "std": float(std)
            }
        )

        return predictions, lower, upper, model_info