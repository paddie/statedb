package statedb

// --------------------------
// Immutable Data Structures
// --------------------------
// TypeMap is the type used for the immutable and mutable fields
// - unabstracted type: map[type string][key Key]*State
type ImmKeyTypeMap map[string]ImmStateMap

// Map from Key (string or int) to *State
type ImmStateMap map[Key]*ImmState

type ImmState struct {
	KT  KeyType // save keytype to match against in testing
	Val []byte  // serialised object
}

func (m ImmKeyTypeMap) count() int {

	if len(m) == 0 {
		return 0
	}

	cnt := 0
	for _, v := range m {
		cnt += len(v)
	}
	return cnt
}

func (m ImmKeyTypeMap) contains(kt *KeyType) bool {
	if sm, ok := m[kt.TypeID()]; ok {
		_, ok = sm[*kt.Key()]
		return ok
	}
	return false
}

func (m ImmKeyTypeMap) lookup(kt *KeyType) *ImmState {
	if sm, ok := m[kt.TypeID()]; ok {
		return sm[*kt.Key()]
	}
	return nil
}

func (m ImmKeyTypeMap) insert(kt *KeyType, val []byte) {

	// allocate KeyMap if nil
	if _, ok := m[kt.TypeID()]; !ok {
		m[kt.TypeID()] = make(ImmStateMap)
	}

	m[kt.TypeID()][kt.K] = &ImmState{
		KT:  *kt,
		Val: val,
	}
}

func (m ImmKeyTypeMap) remove(kt *KeyType) {

	if sm, ok := m[kt.T]; ok {
		delete(sm, kt.K)

		// clen up if there are no more items of this type
		if len(m[kt.T]) == 0 {
			delete(m, kt.T)
		}
	}
}
