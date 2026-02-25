package itemref

import "strings"

func coreIsEmpty(in Ref) bool {
	return in.String() == ""
}

func coreRefType(in Ref) RefVersion {
	v := in.String()
	if strings.HasPrefix(v, "encv1://") {
		return RefVersionEncryptedV1
	}

	return RefVersionPlain
}
