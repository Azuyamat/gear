package gear

import "fmt"

type ValidatedArgs struct {
	args map[string]validatedArg
}
type validatedArg interface{}

func newValidatedArgs() *ValidatedArgs {
	return &ValidatedArgs{
		args: make(map[string]validatedArg),
	}
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
