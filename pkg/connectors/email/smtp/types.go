package smtp

// SendInput is the input for sending an email.
type SendInput struct {
	ToEmail   string   `json:"to_email"`
	Subject   string   `json:"subject"`
	Body      string   `json:"body"`
	RequestID string   `json:"request_id,omitempty"`
	CC        []string `json:"cc,omitempty"`
	BCC       []string `json:"bcc,omitempty"`
	ReplyTo   string   `json:"reply_to,omitempty"`
}

// SendResult is the result of sending an email.
type SendResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	MessageID string `json:"messageId,omitempty"`
}
