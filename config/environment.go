package oglconfig

import (
	"errors"
	"fmt"
)

type Environment string

const (
	// EnvironmentDevelopment is a Environment of type development.
	EnvironmentDevelopment Environment = "development"
	// EnvironmentStaging is a Environment of type staging.
	EnvironmentStaging Environment = "staging"
	// EnvironmentProduction is a Environment of type production.
	EnvironmentProduction Environment = "production"
	// EnvironmentTesting is a Environment of type testing.
	EnvironmentTesting Environment = "testing"
)

var ErrInvalidEnvironment = errors.New("not a valid Environment")

// EnvironmentValues returns a list of the values for Environment
func EnvironmentValues() []Environment {
	return []Environment{
		EnvironmentDevelopment,
		EnvironmentStaging,
		EnvironmentProduction,
		EnvironmentTesting,
	}
}

// String implements the Stringer interface.
func (x Environment) String() string {
	return string(x)
}

// IsValid provides a quick way to determine if the typed value is
// part of the allowed enumerated values
func (x Environment) IsValid() bool {
	_, err := ParseEnvironment(string(x))
	return err == nil
}

var _EnvironmentValue = map[string]Environment{
	"development": EnvironmentDevelopment,
	"staging":     EnvironmentStaging,
	"production":  EnvironmentProduction,
	"testing":     EnvironmentTesting,
}

// ParseEnvironment attempts to convert a string to a Environment.
func ParseEnvironment(name string) (Environment, error) {
	if x, ok := _EnvironmentValue[name]; ok {
		return x, nil
	}

	return Environment(""), fmt.Errorf("%s is %w", name, ErrInvalidEnvironment)
}

// MarshalText implements the text marshaller method.
func (x Environment) MarshalText() ([]byte, error) {
	return []byte(string(x)), nil
}

// UnmarshalText implements the text unmarshaller method.
func (x *Environment) UnmarshalText(text []byte) error {
	tmp, err := ParseEnvironment(string(text))
	if err != nil {
		return err
	}
	*x = tmp

	return nil
}

// AppendText appends the textual representation of itself to the end of b
// (allocating a larger slice if necessary) and returns the updated slice.
//
// Implementations must not retain b, nor mutate any bytes within b[:len(b)].
func (x *Environment) AppendText(b []byte) ([]byte, error) {
	return append(b, x.String()...), nil
}

func (e Environment) IsDev() bool {
	return e == EnvironmentDevelopment
}
