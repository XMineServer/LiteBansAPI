package logging

import "fmt"

type LogFormat int

const (
	LogFormatUnknown LogFormat = iota
	LogFormatText
	LogFormatJSON
)

var logFormatNames = map[LogFormat]string{
	LogFormatText: "text",
	LogFormatJSON: "json",
}

var logFormatValues = map[string]LogFormat{
	"text": LogFormatText,
	"json": LogFormatJSON,
}

func (f LogFormat) String() string {
	if name, ok := logFormatNames[f]; ok {
		return name
	}
	return fmt.Sprintf("LogFormat(%d)", int(f))
}

func (f LogFormat) IsValid() bool {
	_, ok := logFormatNames[f]
	return ok
}

func ParseLogFormat(s string) (LogFormat, error) {
	if f, ok := logFormatValues[s]; ok {
		return f, nil
	}
	return LogFormatUnknown, fmt.Errorf("invalid log format: %q", s)
}

func (f LogFormat) MarshalText() ([]byte, error) {
	if !f.IsValid() {
		return nil, fmt.Errorf("invalid log format: %d", int(f))
	}
	return []byte(f.String()), nil
}

func (f *LogFormat) UnmarshalText(b []byte) error {
	v, err := ParseLogFormat(string(b))
	if err != nil {
		return err
	}
	*f = v
	return nil
}
