package codec

import "errors"

type Noop struct{}

func NewNoop() *Noop {
	return &Noop{}
}

func (n *Noop) Encode(value string) (string, error) {
	return value, nil
}

func (n *Noop) Decode(value string) (string, error) {
	return value, nil
}

func (n *Noop) ExportKey() (string, error) {
	return "", nil
}

func (n *Noop) ImportKey(_ any) error {
	return errors.New("Noop codec does not support key import")
}
