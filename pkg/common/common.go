package common

import (
	"fmt"
	"regexp"
	"strings"
)

func ConvertLabelsStringToMap(labels string) (map[string]string, error) {
	const labelsDelimiter = ","
	const labelsKeyValueDelimiter = "="

	labelsMap := make(map[string]string)
	if labels == "" {
		return labelsMap, nil
	}

	regexKey, _ := regexp.Compile(`^\p{Ll}[\p{Ll}0-9_-]{0,62}$`)
	checkLabelKeyFn := func(key string) error {
		if !regexKey.MatchString(key) {
			return fmt.Errorf("label value %q is invalid (should start with lowercase letter / lowercase letter, digit, _ and - chars are allowed / 1-63 characters", key)
		}
		return nil
	}

	regexValue, _ := regexp.Compile(`^[\p{Ll}0-9_-]{0,63}$`)
	checkLabelValueFn := func(value string) error {
		if !regexValue.MatchString(value) {
			return fmt.Errorf("label value %q is invalid (lowercase letter, digit, _ and - chars are allowed / 0-63 characters", value)
		}

		return nil
	}

	keyValueStrings := strings.Split(labels, labelsDelimiter)
	for _, keyValue := range keyValueStrings {
		keyValue := strings.Split(keyValue, labelsKeyValueDelimiter)

		if len(keyValue) != 2 {
			return nil, fmt.Errorf("labels %q are invalid, correct format: 'key1=value1,key2=value2'", labels)
		}

		key := strings.TrimSpace(keyValue[0])
		if err := checkLabelKeyFn(key); err != nil {
			return nil, err
		}

		value := strings.TrimSpace(keyValue[1])
		if err := checkLabelValueFn(value); err != nil {
			return nil, err
		}

		labelsMap[key] = value
	}

	const maxNumberOfLabels = 64
	if len(labelsMap) > maxNumberOfLabels {
		return nil, fmt.Errorf("more than %d labels is not allowed, given: %d", maxNumberOfLabels, len(labelsMap))
	}

	return labelsMap, nil
}
