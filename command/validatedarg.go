package command

import "fmt"

type ValidatedArgs struct {
	args  map[string]validatedArg
	flags map[string]validatedArg
}
type validatedArg interface{}

func newValidatedArgs() *ValidatedArgs {
	return &ValidatedArgs{
		args:  make(map[string]validatedArg),
		flags: make(map[string]validatedArg),
	}
}

func (v *ValidatedArgs) set(name string, value validatedArg) {
	v.args[name] = value
}

func (v *ValidatedArgs) setFlag(name string, value validatedArg) {
	v.flags[name] = value
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
