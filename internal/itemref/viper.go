package itemref

import (
	"fmt"

	"github.com/spf13/viper"
)

type viperCfg struct {
	val string
}

func (v *viperCfg) Version() RefVersion {
	return coreRefType(v)
}

func (v *viperCfg) IsEmpty() bool {
	return coreIsEmpty(v)
}

func (v *viperCfg) String() string {
	return v.val
}

func getViper(key string) (*viperCfg, error) {
	v := viper.GetString(key)
	if v == "" {
		return nil, fmt.Errorf("viper: missing %s", key)
	}

	return &viperCfg{val: v}, nil
}
