package statedb

// --------------------------
// Delta Data Structures
// --------------------------
type DeltaTypeMap map[string]DeltaStateOpMap

type DeltaStateOpMap map[Key]*StateOp

type StateOp struct {
	KT     KeyType
	Action int // DELETE=-1 or CREATE=1
	Val    []byte
}

func (m DeltaTypeMap) count() int {

	if len(m) == 0 {
		return 0
	}

	cnt := 0
	for _, v := range m {
		cnt += len(v)
	}
	return cnt
}

func (m DeltaTypeMap) contains(kt *KeyType) bool {
	if sm, ok := m[kt.TypeID()]; ok {
		_, ok = sm[*kt.Key()]
		return ok
	}
	return false
}

func (m DeltaTypeMap) lookup(kt *KeyType) *StateOp {
	if sm, ok := m[kt.TypeID()]; ok {
		return sm[*kt.Key()]
	}
	return nil
}

func (m DeltaTypeMap) insert(kt *KeyType, val []byte) {

	// allocate KeyMap if nil
	if _, ok := m[kt.TypeID()]; !ok {
		m[kt.TypeID()] = make(DeltaStateOpMap)
	}

	m[kt.TypeID()][kt.K] = &StateOp{
		KT:     *kt,
		Val:    val,
		Action: INSERT,
	}
}

func (m DeltaTypeMap) remove(kt *KeyType) {
	if _, ok := m[kt.TypeID()]; !ok {
		m[kt.TypeID()] = make(DeltaStateOpMap)
	}

	if _, ok := m[kt.T][kt.K]; ok {
		// delet the entry if it already exists
		delete(m[kt.T], kt.K)
		// if there are no more objects of this type;
		// delete the type entry
		// - prevents creating empty maps
		if len(m[kt.T]) == 0 {
			delete(m, kt.T)
		}
	} else {
		// insert a DELETE entry for this keytype
		m[kt.T][kt.K] = &StateOp{
			KT:     *kt,
			Action: REMOVE,
			Val:    nil,
		}
	}
}
