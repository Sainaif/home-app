"""
Data models for ML forecasting service
"""
from datetime import datetime
from enum import Enum
from typing import List, Optional

from pydantic import BaseModel, Field


class TargetType(str, Enum):
    """Forecast target types"""
    ELECTRICITY = "electricity"
    GAS = "gas"
    SHARED_BUDGET = "shared_budget"


class ModelType(str, Enum):
    """Available forecasting models"""
    SARIMAX = "SARIMAX"
    HOLT_WINTERS = "Holt-Winters"
    SIMPLE_ES = "Simple Exponential Smoothing"
    MOVING_AVERAGE = "Moving Average"


class ForecastRequest(BaseModel):
    """Request for generating a forecast"""
    target: TargetType = Field(..., description="Type of forecast target")
    historical_dates: List[str] = Field(..., description="Historical dates (ISO format)")
    historical_values: List[float] = Field(..., description="Historical values (units)")
    horizon_months: int = Field(default=3, ge=1, le=12, description="Forecast horizon in months")
    confidence_level: float = Field(default=0.95, ge=0.5, le=0.99, description="Confidence interval level")
    cost_per_unit: Optional[float] = Field(None, description="Optional cost per unit for PLN calculation")


class ModelInfo(BaseModel):
    """Information about the forecasting model used"""
    name: ModelType
    version: str = "1.0"
    parameters: dict = Field(default_factory=dict)
    fit_stats: dict = Field(default_factory=dict, description="Model fit statistics (AIC, etc.)")


class ConfidenceInterval(BaseModel):
    """Confidence interval for predictions"""
    lower: List[float] = Field(..., description="Lower bound")
    upper: List[float] = Field(..., description="Upper bound")


class ForecastResponse(BaseModel):
    """Response containing forecast results"""
    target: TargetType
    model: ModelInfo
    predicted_dates: List[str] = Field(..., description="Forecast dates (ISO format)")
    predicted_values: List[float] = Field(..., description="Predicted values (units)")
    confidence_interval: ConfidenceInterval
    predicted_costs: Optional[List[float]] = Field(None, description="Predicted costs (PLN) if cost_per_unit provided")
    created_at: str = Field(default_factory=lambda: datetime.now().isoformat())