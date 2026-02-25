package itemref

type encrypted struct {
	val string
}

func (e *encrypted) Version() RefVersion {
	return RefVersionEncryptedV1
}

func (e *encrypted) IsEmpty() bool {
	return coreIsEmpty(e)
}

func (e *encrypted) String() string {
	return e.val
}
