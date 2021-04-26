package httpx

// Code is an error code.
type Code int

const (
	// Code specifically for broadcast service.
	ErrReadMessage Code = iota + 10000
	ErrIncorrectMetadata
	ErrMetadataNotMatched
	ErrFailedToCreateSubscriber

	// Code for Common errors.
	ErrUnmarshalJSON
)

// Errors maps error code to error message.
var Errors = map[Code]string{
	ErrReadMessage:              "Could not read message",
	ErrIncorrectMetadata:        "Incorrect edge device metadata",
	ErrMetadataNotMatched:       "Metadata not matched with any existing session",
	ErrFailedToCreateSubscriber: "Failed to create subscriber for user",
	ErrUnmarshalJSON:            "Could not unmarshal JSON data",
}
