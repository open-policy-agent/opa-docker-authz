// Copyright 2018 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package topdown

import (
	"errors"
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"
)

var nothingResolver ast.Resolver = illegalResolver{}

func builtinOPARuntime(bctx BuiltinContext, _ []*ast.Term, iter func(*ast.Term) error) error {

	if bctx.Runtime == nil {
		return iter(ast.InternedEmptyObject)
	}

	if bctx.Runtime.Get(ast.InternedTerm("config")) != nil {
		iface, err := ast.ValueToInterface(bctx.Runtime.Value, nothingResolver)
		if err != nil {
			return err
		}
		if object, ok := iface.(map[string]any); ok {
			if cfgRaw, ok := object["config"]; ok {
				if config, ok := cfgRaw.(map[string]any); ok {
					configPurged, err := activeConfig(config)
					if err != nil {
						return err
					}
					object["config"] = configPurged
					value, err := ast.InterfaceToValue(object)
					if err != nil {
						return err
					}
					return iter(ast.NewTerm(value))
				}
			}
		}
	}

	return iter(bctx.Runtime)
}

func init() {
	RegisterBuiltinFunc(ast.OPARuntime.Name, builtinOPARuntime)
}

func activeConfig(config map[string]any) (any, error) {

	if config["services"] != nil {
		err := removeServiceCredentials(config["services"])
		if err != nil {
			return nil, err
		}
	}

	if config["keys"] != nil {
		err := removeCryptoKeys(config["keys"])
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func removeServiceCredentials(x any) error {

	switch x := x.(type) {
	case []any:
		for _, v := range x {
			err := removeKey(v, "credentials")
			if err != nil {
				return err
			}
		}

	case map[string]any:
		for _, v := range x {
			err := removeKey(v, "credentials")
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("illegal service config type: %T", x)
	}

	return nil
}

func removeCryptoKeys(x any) error {

	switch x := x.(type) {
	case map[string]any:
		for _, v := range x {
			err := removeKey(v, "key", "private_key")
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("illegal keys config type: %T", x)
	}

	return nil
}

func removeKey(x any, keys ...string) error {
	val, ok := x.(map[string]any)
	if !ok {
		return errors.New("type assertion error")
	}

	for _, key := range keys {
		delete(val, key)
	}

	return nil
}

type illegalResolver struct{}

func (illegalResolver) Resolve(ref ast.Ref) (any, error) {
	return nil, fmt.Errorf("illegal value: %v", ref)
}
