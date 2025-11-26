package command

import "fmt"

type ValidatedArgs struct {
	args     map[string]validatedArg
	flags    map[string]validatedArg
	variadic map[string][]interface{}
}
type validatedArg interface{}

func newValidatedArgs() *ValidatedArgs {
	return &ValidatedArgs{
		args:     make(map[string]validatedArg),
		flags:    make(map[string]validatedArg),
		variadic: make(map[string][]interface{}),
	}
}

func (v *ValidatedArgs) set(name string, value validatedArg) {
	v.args[name] = value
}

func (v *ValidatedArgs) setFlag(name string, value validatedArg) {
	v.flags[name] = value
}

func (v *ValidatedArgs) setVariadic(name string, values []interface{}) {
	v.variadic[name] = values
}

func (v *ValidatedArgs) Get(name string) validatedArg {
	return v.args[name]
}

func (v *ValidatedArgs) Has(name string) bool {
	_, ok := v.args[name]
	return ok
}

func (v *ValidatedArgs) String(name string) string {
	str, _ := v.GetString(name)
	return str
}

func (v *ValidatedArgs) GetString(name string) (string, error) {
	arg, ok := v.args[name]
	if !ok {
		return "", fmt.Errorf("argument %s not found", name)
	}
	str, ok := arg.(string)
	if !ok {
		return "", fmt.Errorf("argument %s is not a string", name)
	}
	return str, nil
}

func (v *ValidatedArgs) Int(name string) int {
	i, _ := v.GetInt(name)
	return i
}

func (v *ValidatedArgs) GetInt(name string) (int, error) {
	arg, ok := v.args[name]
	if !ok {
		return 0, fmt.Errorf("argument %s not found", name)
	}
	i, ok := arg.(int)
	if !ok {
		return 0, fmt.Errorf("argument %s is not an int", name)
	}
	return i, nil
}

func (v *ValidatedArgs) Float(name string) float64 {
	f, _ := v.GetFloat(name)
	return f
}

func (v *ValidatedArgs) GetFloat(name string) (float64, error) {
	arg, ok := v.args[name]
	if !ok {
		return 0, fmt.Errorf("argument %s not found", name)
	}
	f, ok := arg.(float64)
	if !ok {
		return 0, fmt.Errorf("argument %s is not a float", name)
	}
	return f, nil
}

func (v *ValidatedArgs) Bool(name string) bool {
	b, _ := v.GetBool(name)
	return b
}

func (v *ValidatedArgs) GetBool(name string) (bool, error) {
	arg, ok := v.args[name]
	if !ok {
		return false, fmt.Errorf("argument %s not found", name)
	}
	b, ok := arg.(bool)
	if !ok {
		return false, fmt.Errorf("argument %s is not a bool", name)
	}
	return b, nil
}

func (v *ValidatedArgs) GetFlag(name string) validatedArg {
	return v.flags[name]
}

func (v *ValidatedArgs) HasFlag(name string) bool {
	_, ok := v.flags[name]
	return ok
}

func (v *ValidatedArgs) FlagString(name string) string {
	str, _ := v.GetFlagString(name)
	return str
}

func (v *ValidatedArgs) GetFlagString(name string) (string, error) {
	flag, ok := v.flags[name]
	if !ok {
		return "", fmt.Errorf("flag %s not found", name)
	}
	str, ok := flag.(string)
	if !ok {
		return "", fmt.Errorf("flag %s is not a string", name)
	}
	return str, nil
}

func (v *ValidatedArgs) FlagInt(name string) int {
	i, _ := v.GetFlagInt(name)
	return i
}

func (v *ValidatedArgs) GetFlagInt(name string) (int, error) {
	flag, ok := v.flags[name]
	if !ok {
		return 0, fmt.Errorf("flag %s not found", name)
	}
	i, ok := flag.(int)
	if !ok {
		return 0, fmt.Errorf("flag %s is not an int", name)
	}
	return i, nil
}

func (v *ValidatedArgs) FlagFloat(name string) float64 {
	f, _ := v.GetFlagFloat(name)
	return f
}

func (v *ValidatedArgs) GetFlagFloat(name string) (float64, error) {
	flag, ok := v.flags[name]
	if !ok {
		return 0, fmt.Errorf("flag %s not found", name)
	}
	f, ok := flag.(float64)
	if !ok {
		return 0, fmt.Errorf("flag %s is not a float", name)
	}
	return f, nil
}

func (v *ValidatedArgs) FlagBool(name string) bool {
	b, _ := v.GetFlagBool(name)
	return b
}

func (v *ValidatedArgs) GetFlagBool(name string) (bool, error) {
	flag, ok := v.flags[name]
	if !ok {
		return false, fmt.Errorf("flag %s not found", name)
	}
	b, ok := flag.(bool)
	if !ok {
		return false, fmt.Errorf("flag %s is not a bool", name)
	}
	return b, nil
}

func (v *ValidatedArgs) GetVariadic(name string) []interface{} {
	return v.variadic[name]
}

func (v *ValidatedArgs) HasVariadic(name string) bool {
	_, ok := v.variadic[name]
	return ok
}

func (v *ValidatedArgs) GetVariadicStrings(name string) ([]string, error) {
	values, ok := v.variadic[name]
	if !ok {
		return nil, fmt.Errorf("variadic argument %s not found", name)
	}
	result := make([]string, len(values))
	for i, val := range values {
		str, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("variadic argument %s at position %d is not a string", name, i)
		}
		result[i] = str
	}
	return result, nil
}

func (v *ValidatedArgs) VariadicStrings(name string) []string {
	strs, _ := v.GetVariadicStrings(name)
	return strs
}

func (v *ValidatedArgs) GetVariadicInts(name string) ([]int, error) {
	values, ok := v.variadic[name]
	if !ok {
		return nil, fmt.Errorf("variadic argument %s not found", name)
	}
	result := make([]int, len(values))
	for i, val := range values {
		num, ok := val.(int)
		if !ok {
			return nil, fmt.Errorf("variadic argument %s at position %d is not an int", name, i)
		}
		result[i] = num
	}
	return result, nil
}

func (v *ValidatedArgs) VariadicInts(name string) []int {
	ints, _ := v.GetVariadicInts(name)
	return ints
}

func (v *ValidatedArgs) GetVariadicFloats(name string) ([]float64, error) {
	values, ok := v.variadic[name]
	if !ok {
		return nil, fmt.Errorf("variadic argument %s not found", name)
	}
	result := make([]float64, len(values))
	for i, val := range values {
		num, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("variadic argument %s at position %d is not a float", name, i)
		}
		result[i] = num
	}
	return result, nil
}

func (v *ValidatedArgs) VariadicFloats(name string) []float64 {
	floats, _ := v.GetVariadicFloats(name)
	return floats
}

func (v *ValidatedArgs) GetVariadicBools(name string) ([]bool, error) {
	values, ok := v.variadic[name]
	if !ok {
		return nil, fmt.Errorf("variadic argument %s not found", name)
	}
	result := make([]bool, len(values))
	for i, val := range values {
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("variadic argument %s at position %d is not a bool", name, i)
		}
		result[i] = b
	}
	return result, nil
}

func (v *ValidatedArgs) VariadicBools(name string) []bool {
	bools, _ := v.GetVariadicBools(name)
	return bools
}
