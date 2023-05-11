package data

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// The AdditionalInfo struct represents the data in the JSON/JSONB column.
// It implements driver.Valuer and sql.Scanner interfaces.
type AdditionalInfo struct {
	FileName      string
	FileExtension string
	FileSize      string
}

// This method implements the driver.Valuer interface and
// simply returns the JSON-encoded representation of the struct
func (a AdditionalInfo) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// This method implements the sql.Scanner interface and
// simply decodes a JSON-encoded value into the struct fields.
func (a *AdditionalInfo) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}
