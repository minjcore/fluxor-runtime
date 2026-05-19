package zalo

import (
	"encoding/json"
	"regexp"
	"strconv"
)

// NormalizePhone converts Vietnamese phone formats to standard 84xxxxxxxxx.
// Examples: 0898527311 -> 84898527311, +84898527311 -> 84898527311.
func NormalizePhone(phone string) string {
	if phone == "" {
		return ""
	}
	re := regexp.MustCompile(`\D`)
	digits := re.ReplaceAllString(phone, "")
	if len(digits) >= 10 && digits[0] == '0' {
		digits = "84" + digits[1:]
	} else if len(digits) > 11 && digits[:2] == "84" {
		digits = "84" + digits[len(digits)-9:]
	} else if len(digits) == 9 {
		digits = "84" + digits
	}
	return digits
}

// flexInt unmarshals from JSON as either number or string (Zalo API may return expires_in as string).
type flexInt int

func (fi *flexInt) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		*fi = flexInt(n)
		return nil
	}
	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*fi = flexInt(n)
	return nil
}

// TokenResponse is the response from Zalo OAuth token endpoints (exchange code / refresh).
type TokenResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	ExpiresIn    flexInt `json:"expires_in"`
	Error        int     `json:"error,omitempty"`
	Message      string  `json:"message,omitempty"`
}

// SendZNSInput is the input for sending a ZNS templated message.
type SendZNSInput struct {
	Phone       string            `json:"phone"`
	TemplateID  string            `json:"template_id"`
	TemplateData map[string]interface{} `json:"template_data"`
	TrackingID  string            `json:"tracking_id,omitempty"`
}

// SendZNSResult is the result of sending a ZNS message.
type SendZNSResult struct {
	MsgID    string `json:"msg_id,omitempty"`
	SentTime int64  `json:"sent_time,omitempty"`
	Success  bool   `json:"success"`
	Error    int    `json:"error,omitempty"`
	Message  string `json:"message,omitempty"`
}

// TemplateInfo is ZNS template information from the template/info API.
type TemplateInfo struct {
	TemplateID   string                 `json:"template_id"`
	Name         string                 `json:"name,omitempty"`
	Status       int                    `json:"status,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
}

// QuotaInfo is ZNS quota information from the message/quota API.
type QuotaInfo struct {
	Quota        int64                  `json:"quota,omitempty"`
	Used         int64                  `json:"used,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
}
