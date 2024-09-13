package vm

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/stretchr/testify/assert"
)

func TestInterpreterFactory(t *testing.T) {
	evm := &EVM{}

	interpreter := NewInterpreter("", evm)
	if interpreter == nil {
		t.Error("Expected interpreter to be created, but got nil")
	}
	interpreter = NewInterpreter("geth", evm)
	if interpreter == nil {
		t.Error("Expected interpreter to be created, but got nil")
	}
	asserted := assert.Panics(t, func() { NewInterpreter("invalid", evm) })
	if !asserted {
		t.Error("Expected panic with invalid interpreter name")
	}
}

func TestInterpreterFactory_RegisterInterpreter(t *testing.T) {
	evm := &EVM{}
	correctFactoryIsCalled := false

	RegisterInterpreterFactory("newInterpreter", func(evm *EVM) Interpreter {
		correctFactoryIsCalled = true
		return NewEVMInterpreter(evm)
	})

	interpreter := NewInterpreter("newInterpreter", evm)
	if interpreter == nil {
		t.Error("Expected interpreter to be created, but got nil")
	}
	if !correctFactoryIsCalled {
		t.Error("Wrong interpreter factory has been called")
	}
}

func TestInterpreterRegistry_NameCollisionsAreDetected(t *testing.T) {
	const testInterpreter = "test-evm-001"
	factory := func(evm *EVM) Interpreter { return nil }
	err := RegisterInterpreterFactory(testInterpreter, factory)
	if err != nil {
		t.Error("Expected no error when registering new interpreter")
	}
	err = RegisterInterpreterFactory(testInterpreter, factory)
	if !errors.Is(err, ErrInterpreterNameCollision) {
		t.Errorf("Expected error %v, but got %v", ErrInterpreterNameCollision, err)
	}
}

func TestGetInterpreter_ProducesGethInstanceWhenTracingIsEnabled(t *testing.T) {
	const testInterpreter = "test-evm-002"
	evm := &EVM{}
	evm.Config.InterpreterImpl = testInterpreter

	RegisterInterpreterFactory(testInterpreter, func(evm *EVM) Interpreter {
		return nil
	})

	// Without tracer, the requested interpreter is provided.
	interpreter := getInterpreter(evm)
	if _, ok := interpreter.(*EVMInterpreter); ok {
		t.Errorf("Expected interpreter from factory, but got Geth interpreter")
	}

	// With tracer, the Geth interpreter is provided.
	evm.Config.Tracer = &tracing.Hooks{}
	interpreter = getInterpreter(evm)
	if _, ok := interpreter.(*EVMInterpreter); !ok {
		t.Error("Expected Geth interpreter to be created, but got different")
	}
}
