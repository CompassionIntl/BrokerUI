package structs

import "time"

type StandardMessage struct {
	MessageID string
	Timestamp time.Time
	Headers   map[string]string
	Body      string
}

type RequestMessageIDs struct {
	MessageIDs []string `json:"messageIDs"`
}
