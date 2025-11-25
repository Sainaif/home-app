package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type OCRResult struct {
	InvoiceNumber string `json:"invoice_number"`
	Date          string `json:"date"`
	TotalBrutto   string `json:"total_brutto"`
	Deadline      string `json:"deadline"`
	SellersName   string `json:"sellers_name"`
	Units         string `json:"units"`
	BillType      string `json:"bill_type"`
	PeriodFrom    string `json:"period_from"`
	PeriodTo      string `json:"period_to"`
}

type OCRService struct {
	apiKey string
}

func NewOCRService() *OCRService {
	return &OCRService{
		apiKey: os.Getenv("OPENAI_API_KEY"),
	}
}

func (s *OCRService) ParseInvoice(ctx context.Context, imageBytes []byte) (*OCRResult, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	imgBase64 := base64.StdEncoding.EncodeToString(imageBytes)

	body := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []interface{}{
			map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": `Extract the following information from this invoice and return ONLY a valid JSON object with these exact fields:
- invoice_number: invoice/bill number
- date: invoice date (DD.MM.YYYY format)
- total_brutto: total amount with currency (e.g., "909,78 zł")
- deadline: payment deadline (DD.MM.YYYY format)
- sellers_name: name of the seller/company
- units: total usage/consumption units (e.g., "150 kWh" for electricity, "100 m³" for gas). If not found, return empty string.
- bill_type: determine the type of bill based on content. Return one of: "electricity" (for prąd/energia elektryczna/kWh), "gas" (for gaz/m³), "internet" (for internet/telefon), or "inne" (for anything else)
- period_from: SETTLEMENT PERIOD start date (DD.MM.YYYY format) - look for "Okres rozliczeniowy od" or "settlement period from" or the consumption period start. NOT the contract period. If not found, return empty string.
- period_to: SETTLEMENT PERIOD end date (DD.MM.YYYY format) - look for "Okres rozliczeniowy do" or "settlement period to" or the consumption period end. NOT the contract period. If not found, return empty string.

IMPORTANT: For period_from and period_to, look for the actual billing/settlement/consumption period (when the service was used), NOT the contract validity period or subscription dates.

Do not include any markdown formatting or code blocks, just the raw JSON.`,
					},
					map[string]interface{}{
						"type": "image_url",
						"image_url": map[string]string{
							"url": "data:image/jpeg;base64," + imgBase64,
						},
					},
				},
			},
		},
		"max_tokens": 500,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract the content from OpenAI's response format
	choices, ok := parsed["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("invalid response format: no choices")
	}

	firstChoice, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid choice format")
	}

	message, ok := firstChoice["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid message format")
	}

	content, ok := message["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid content format")
	}

	// Parse the JSON from the content
	var result OCRResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse OCR result: %w (content: %s)", err, content)
	}

	return &result, nil
}
