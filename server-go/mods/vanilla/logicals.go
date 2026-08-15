package main

import (
	"fmt"

	"game/server/mods/shared"
)

type AdvancedOperation string

const (
	AdvancedAnd AdvancedOperation = "and"
	AdvancedOr  AdvancedOperation = "or"
	AdvancedNot AdvancedOperation = "not"
	AdvancedIs  AdvancedOperation = "is"
)

type advancedValueKind uint8

const (
	advancedStringValue advancedValueKind = iota
	advancedExpressionValue
)

type AdvancedValue struct {
	kind advancedValueKind

	stringValue     string
	expressionValue *AdvancedExpression
}

type AdvancedExpression struct {
	Operation AdvancedOperation
	Values    []AdvancedValue
}

func StringValue(value string) AdvancedValue {
	return AdvancedValue{
		kind:        advancedStringValue,
		stringValue: value,
	}
}

func ExpressionValue(value AdvancedExpression) AdvancedValue {
	return AdvancedValue{
		kind:              advancedExpressionValue,
		expressionValue: &value,
	}
}

func isAdvanced(
	state *shared.GameState,
	expression AdvancedExpression,
) bool {
	result, err := evaluateAdvanced(state, expression)
	if err != nil {
		return false
	}

	return result
}

func evaluateAdvanced(
	state *shared.GameState,
	expression AdvancedExpression,
) (bool, error) {
	switch expression.Operation {
	case AdvancedIs:
		return evaluateIs(state, expression)

	case AdvancedAnd:
		return evaluateAnd(state, expression)

	case AdvancedOr:
		return evaluateOr(state, expression)

	case AdvancedNot:
		return evaluateNot(state, expression)

	default:
		return false, fmt.Errorf(
			"unknown advanced operation %q",
			expression.Operation,
		)
	}
}

func evaluateIs(
	state *shared.GameState,
	expression AdvancedExpression,
) (bool, error) {
	if len(expression.Values) != 2 {
		return false, fmt.Errorf(
			`"is" requires exactly two string values`,
		)
	}

	layer1Key, err := getStringValue(expression.Values[0])
	if err != nil {
		return false, fmt.Errorf(
			`"is" layer key: %w`,
			err,
		)
	}

	childKey, err := getStringValue(expression.Values[1])
	if err != nil {
		return false, fmt.Errorf(
			`"is" child key: %w`,
			err,
		)
	}

	currentIdx := getSelection(state, layer1Key)
	targetIdx := indexOfChild(
		getBranch(state, layer1Key),
		childKey,
	)

	return currentIdx >= targetIdx, nil
}

func evaluateAnd(
	state *shared.GameState,
	expression AdvancedExpression,
) (bool, error) {
	if len(expression.Values) == 0 {
		return false, fmt.Errorf(
			`"and" requires at least one expression`,
		)
	}

	for _, value := range expression.Values {
		result, err := evaluateExpressionValue(state, value)
		if err != nil {
			return false, err
		}

		if !result {
			return false, nil
		}
	}

	return true, nil
}

func evaluateOr(
	state *shared.GameState,
	expression AdvancedExpression,
) (bool, error) {
	if len(expression.Values) == 0 {
		return false, fmt.Errorf(
			`"or" requires at least one expression`,
		)
	}

	for _, value := range expression.Values {
		result, err := evaluateExpressionValue(state, value)
		if err != nil {
			return false, err
		}

		if result {
			return true, nil
		}
	}

	return false, nil
}

func evaluateNot(
	state *shared.GameState,
	expression AdvancedExpression,
) (bool, error) {
	if len(expression.Values) != 1 {
		return false, fmt.Errorf(
			`"not" requires exactly one expression`,
		)
	}

	value := expression.Values[0]

	if value.kind != advancedExpressionValue ||
		value.expressionValue == nil {
		return false, fmt.Errorf(
			`"not" requires an expression value`,
		)
	}

	result, err := evaluateAdvanced(state, *value.expressionValue)
	if err != nil {
		return false, err
	}

	return !result, nil
}

func evaluateExpressionValue(
	state *shared.GameState,
	value AdvancedValue,
) (bool, error) {
	if value.kind != advancedExpressionValue ||
		value.expressionValue == nil {
		return false, fmt.Errorf(
			`"and" and "or" require expression values`,
		)
	}

	return evaluateAdvanced(state, *value.expressionValue)
}

func getStringValue(value AdvancedValue) (string, error) {
	if value.kind != advancedStringValue {
		return "", fmt.Errorf(
			`expected a string value`,
		)
	}

	return value.stringValue, nil
}

func advIs(layer1Key, childKey string) AdvancedExpression {
	return AdvancedExpression{
		Operation: AdvancedIs,
		Values: []AdvancedValue{
			StringValue(layer1Key),
			StringValue(childKey),
		},
	}
}

func advAnd(expressions ...AdvancedExpression) AdvancedExpression {
	values := make([]AdvancedValue, 0, len(expressions))

	for _, expression := range expressions {
		values = append(values, ExpressionValue(expression))
	}

	return AdvancedExpression{
		Operation: AdvancedAnd,
		Values:    values,
	}
}

func advOr(expressions ...AdvancedExpression) AdvancedExpression {
	values := make([]AdvancedValue, 0, len(expressions))

	for _, expression := range expressions {
		values = append(values, ExpressionValue(expression))
	}

	return AdvancedExpression{
		Operation: AdvancedOr,
		Values:    values,
	}
}

func advNot(expression AdvancedExpression) AdvancedExpression {
	return AdvancedExpression{
		Operation: AdvancedNot,
		Values: []AdvancedValue{
			ExpressionValue(expression),
		},
	}
}
