package data

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// AdditionalInfo implements database/sql.Scanner and database/sql/driver.Valuer interfaces.
// This is to enable the AdditionalInfo field of the Book struct to be saved as jsonb in the database.
type AdditionalInfo map[string]string

// Value marshals the object into a JSON byte slice that can be understood by the database.
func (a AdditionalInfo) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan unmarshals a JSON byte slice from the database into the map.
func (a AdditionalInfo) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch data := value.(type) {
	case string:
		return json.Unmarshal([]byte(data), &a)
	case []byte:
		return json.Unmarshal(data, &a)
	default:
		return fmt.Errorf("type assertion to %t failed", value)
	}
}
