// errkit implements all errors generated by nuclei and includes error definations
// specific to nuclei , error classification (like network,logic) etc
package errkit

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/projectdiscovery/utils/env"
)

const (
	// DelimArrow is delim used by projectdiscovery/utils to join errors
	DelimArrow = "<-"
	// DelimSemiColon is standard delim popularly used to join errors
	DelimSemiColon = "; "
	// DelimMultiLine is delim used to join errors in multiline format
	DelimMultiLine = "\n -  "
	// MultiLinePrefix is the prefix used for multiline errors
	MultiLineErrPrefix = "the following errors occurred:"
)

const (
	// ErrClassNetwork indicates an error related to network operations
	// these may be resolved by retrying the operation with exponential backoff
	// ex: Timeout awaiting headers, connection reset by peer etc
	ErrClassNetworkTemporary = "network-temporary-error"
	// ErrClassNetworkPermanent indicates a permanent error related to network operations
	// these may not be resolved by retrying and need manual intervention
	// ex: no address found for host
	ErrClassNetworkPermanent = "network-permanent-error"
	// ErrClassDeadline indicates a timeout error in logical operations
	// these are custom deadlines set by nuclei itself to prevent infinite hangs
	// and in most cases are server side issues (ex: server connects but does not respond at all)
	// a manual intervention is required
	ErrClassDeadline = "deadline-error"
	// ErrClassLogic indicates an error in logical operations or decision-making
	// these are caused if a dependency is not met ex: next operation requires csrf token
	// but csrf token is not present (these can be safely ignored and does not actually represent an error)
	ErrClassTemplateLogic = "template-logic-error"
	// ErrClassDataMissing indicates an error due to missing required data
	// ex: a template that requires username , password or something else but none were provided
	ErrClassDataMissing = "data-missing"
	// ErrClassUnknown indicates an unknown error class
	ErrClassUnknown = "unknown-class"
)

var (
	// MaxErrorDepth is the maximum depth of errors to be unwrapped or maintained
	// all errors beyond this depth will be ignored
	MaxErrorDepth = env.GetEnvOrDefault("MAX_ERROR_DEPTH", 3)
	// ErrorSeperator is the seperator used to join errors
	ErrorSeperator = env.GetEnvOrDefault("ERROR_SEPERATOR", "; ")
)

// ErrorX is a custom error type that can handle all known types of errors
// wrapping and joining strategies including custom ones and it supports error class
// which can be shown to client/users in more meaningful way
type ErrorX struct {
	class string
	errs  []error
}

// Build returns the object as error interface
func (e *ErrorX) Build() error {
	return e
}

// Unwrap returns the underlying error
func (e *ErrorX) Unwrap() []error {
	return e.errs
}

// Is checks if current error contains given error
func (e *ErrorX) Is(err error) bool {
	x := &ErrorX{}
	parseError(x, err)
	// even one submatch is enough
	for _, orig := range e.errs {
		for _, match := range x.errs {
			if errors.Is(orig, match) {
				return true
			}
		}
	}

	return false
}

// MarshalJSON returns the json representation of the error
func (e *ErrorX) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"class":  e.class,
		"errors": e.errs,
	}
	return json.Marshal(m)
}

// Error returns the error string
func (e *ErrorX) Error() string {
	var sb strings.Builder
	if e.class != "" {
		sb.WriteString("class=")
		sb.WriteString(e.getOriginClass())
		sb.WriteString(" ")
	}
	for _, err := range e.errs {
		sb.WriteString(err.Error())
		sb.WriteString(ErrorSeperator)
	}
	return strings.TrimSuffix(sb.String(), ErrorSeperator)
}

// Cause return the original error that caused this without any wrapping
func (e *ErrorX) Cause() error {
	if len(e.errs) > 0 {
		return e.errs[0]
	}
	return nil
}

// getOriginClass returns the class that was first set
func (e *ErrorX) getOriginClass() string {
	index := strings.LastIndex(e.class, ",")
	if index != -1 {
		return e.class[:index]
	} else {
		return e.class
	}
}

// Class returns the class of the error
// if multiple classes are present, it returns the first one
func (e *ErrorX) Class() string {
	return e.getOriginClass()
}

// FromError parses a given error to understand the error class
// and optionally adds given message for more info
func FromError(err error) *ErrorX {
	if err == nil {
		return nil
	}
	nucleiErr := &ErrorX{}
	parseError(nucleiErr, err)
	return nucleiErr
}

// New creates a new error with the given message
func New(format string, args ...interface{}) *ErrorX {
	return &ErrorX{errs: []error{fmt.Errorf(format, args...)}}
}

// Msgf adds a message to the error
func (e *ErrorX) Msgf(format string, args ...interface{}) {
	if e == nil {
		return
	}
	e.errs = append(e.errs, fmt.Errorf(format, args...))
}

// SetClass sets the class of the error
// if underlying error class was already set, then it is given preference
// when generating final error msg
func (e *ErrorX) SetClass(class string) *ErrorX {
	if e.class != "" {
		e.class = class + "," + e.class
	} else {
		e.class = class
	}
	return e
}

// parseError recursively parses all known types of errors
func parseError(to *ErrorX, err error) {
	if err == nil {
		return
	}
	if to == nil {
		to = &ErrorX{}
	}
	if len(to.errs) >= MaxErrorDepth {
		return
	}

	switch v := err.(type) {
	case *ErrorX:
		to.errs = append(to.errs, v.errs...)
		if to.class == "" {
			to.class = v.class
		} else {
			to.class += "," + v.class
		}
	case JoinedError:
		to.errs = append(to.errs, v.Unwrap()...)
	case WrappedError:
		to.errs = append(to.errs, v.Unwrap())
	case CauseError:
		to.errs = append(to.errs, v.Cause())
		remaining := strings.Replace(err.Error(), v.Cause().Error(), "", -1)
		parseError(to, errors.New(remaining))
	default:
		// try assigning to enriched error
		if strings.Contains(err.Error(), DelimArrow) {
			// Split the error by arrow delim
			parts := strings.Split(err.Error(), DelimArrow)
			for i := len(parts) - 1; i >= 0; i-- {
				part := strings.TrimSpace(parts[i])
				parseError(to, errors.New(part))
			}
		} else if strings.Contains(err.Error(), DelimSemiColon) {
			// Split the error by semi-colon delim
			parts := strings.Split(err.Error(), DelimSemiColon)
			for _, part := range parts {
				part = strings.TrimSpace(part)
				parseError(to, errors.New(part))
			}
		} else if strings.Contains(err.Error(), MultiLineErrPrefix) {
			// remove prefix
			msg := strings.ReplaceAll(err.Error(), MultiLineErrPrefix, "")
			parts := strings.Split(msg, DelimMultiLine)
			for _, part := range parts {
				part = strings.TrimSpace(part)
				parseError(to, errors.New(part))
			}
		} else {
			// this cannot be furthur unwrapped
			to.errs = append(to.errs, err)
		}
	}
}

// WrappedError is implemented by errors that are wrapped
type WrappedError interface {
	// Unwrap returns the underlying error
	Unwrap() error
}
