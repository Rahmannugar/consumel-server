package models

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func EncodeOperationCursor(cursor OperationListCursor) string {
	payload := strconv.FormatInt(cursor.CreatedAt.UnixNano(), 10) + "." + cursor.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeOperationCursor(value string) (*OperationListCursor, error) {
	if value == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrOperationCursorInvalid
	}
	parts := strings.Split(string(decoded), ".")
	if len(parts) != 2 {
		return nil, ErrOperationCursorInvalid
	}
	nanoseconds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, ErrOperationCursorInvalid
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, ErrOperationCursorInvalid
	}
	return &OperationListCursor{CreatedAt: time.Unix(0, nanoseconds).UTC(), ID: id}, nil
}
